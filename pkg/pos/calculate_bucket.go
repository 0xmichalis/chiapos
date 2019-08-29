package pos

import "math"

type F1 struct {
	k      int
	aesKey [32]byte
}

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

// Precomputed shifts that specify which entries match with which other entries
// in adjacent buckets.
var matchingShiftsC = make([][]int, 2)

// Performs the pre-computation of shifts.
func precomputeShifts() {
	for parity := 0; parity < 2; parity++ {
		matchingShiftsC[parity] = make([]int, kC)
		for r := 0; r < kExtraBitsPow; r++ {
			v := int(math.Pow(float64(2*r+parity), 2)) % kC
			matchingShiftsC[parity][r] = v
		}
	}
}
