package pos

import (
	"crypto/aes"
	"crypto/cipher"
	"math"
)

// AES block size
const kBlockSizeBits = 128

// Extra bits of output from the f functions. Instead of being a function from k -> k bits,
// it's a function from k -> k + kExtraBits bits. This allows less collisions in matches.
// Refer to the paper for mathematical motivations.
const kExtraBits = 5

// Convenience variable
const kExtraBitsPow = 1 << kExtraBits

// B and C groups which constitute a bucket, or BC group. These groups determine how
// elements match with each other. Two elements must be in adjacent buckets to match.
const kB = 60
const kC int = 509
const kBC = kB * kC

// This (times k) is the length of the metadata that must be kept for each entry. For example,
// for a table 4 entry, we must keep 4k additional bits for each entry, which is used to
// compute f5.
var kVectorLens = map[int]int{
	2: 1,
	3: 2,
	4: 4,
	5: 4,
	6: 3,
	7: 2,
	8: 0,
}

type F1 struct {
	k   int
	key cipher.Block
}

func NewF1(k int, key []byte) (*F1, error) {
	f1 := &F1{
		k: k,
	}

	aesKey := make([]byte, 32)
	// First byte is 1, the index of this table
	aesKey[0] = 1
	copy(aesKey[1:], key)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	f1.key = block

	precomputeShifts()

	return f1, nil
}

// Precomputed shifts that specify which entries match with which other entries
// in adjacent buckets.
var matchingShiftsC [2][kC]int

// Performs the pre-computation of shifts.
func precomputeShifts() {
	for parity := 0; parity < 2; parity++ {
		for r := 0; r < kExtraBitsPow; r++ {
			v := int(math.Pow(float64(2*r+parity), 2)) % kC
			matchingShiftsC[parity][r] = v
		}
	}
}
