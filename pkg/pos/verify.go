package pos

import (
	"crypto/aes"
	"fmt"
	"math"
	"math/big"
	"math/bits"

	"github.com/kargakis/chiapos/pkg/serialize"
	"github.com/kargakis/chiapos/pkg/utils"
)

// Verify verifies the provided proof given the challenge, seed, and k.
func Verify(challenge string, seed []byte, k int, proof []uint64) error {
	if len(proof) != 64 {
		return fmt.Errorf("invalid proof length: expected 64 values, got %d", len(proof))
	}

	f1, err := NewF1(k, seed)
	if err != nil {
		return err
	}

	var fxs []uint64
	var metadata []*big.Int
	for _, x := range proof {
		fx := f1.CalculateOne(x)
		fxs = append(fxs, fx)
		// TODO: Converting to an int64 (as opposed to uint64) may be problematic for large k?
		metadata = append(metadata, big.NewInt(int64(x)))
	}

	fx, err := NewFx(k, seed)
	if err != nil {
		return err
	}

	for t := 2; t <= 7; t++ {
		var newFxs []uint64
		var newMetadata []*big.Int
		for i := 0; i < int(math.Pow(float64(2), float64(7-t))); i++ {
			leftIndex := i * 2
			rightIndex := leftIndex + 1

			le := []*serialize.Entry{{Fx: fxs[leftIndex]}}
			re := []*serialize.Entry{{Fx: fxs[rightIndex]}}
			if match := FindMatches(le, re); len(match) != 1 {
				return fmt.Errorf("invalid proof: proofs do not match at table %d", t)
			}

			// Then calculate for the next table
			f, err := fx.Calculate(t, fxs[leftIndex], metadata[leftIndex], metadata[rightIndex])
			if err != nil {
				return fmt.Errorf("cannot compute f%d(x): %w", t, err)
			}
			newFxs = append(newFxs, f)
			if t != 7 {
				collated, err := Collate(t, k, metadata[leftIndex], metadata[rightIndex])
				if err != nil {
					return fmt.Errorf("cannot collate outputs: %w", err)
				}
				newMetadata = append(newMetadata, collated)
			}
		}
		fxs = newFxs
		metadata = newMetadata
	}

	// Now truncate both the challenge and the f7 output
	// and see whether the space proof is valid.
	challBig := new(big.Int).SetBytes([]byte(challenge))
	challBig = utils.Trunc(challBig, 0, k, challBig.BitLen())
	target := challBig.Uint64()
	fEntry := utils.TruncPrimitive(fxs[0], 0, k, bits.Len64(fxs[0]))
	if fEntry != target {
		return fmt.Errorf("invalid proof: f7 output does not match the provided challenge")
	}

	return nil
}

// findBucketAndPosForX returns the first item of x's bucket and
// the position of x in the bucket.
func findBucketAndPosForX(x uint64) (uint64, int) {
	return x - (x % (8 * aes.BlockSize)), int(x) % (8 * aes.BlockSize)
}
