package pos

import (
	"crypto/aes"
	"math"

	"github.com/kargakis/chiapos/pkg/utils/bits"
)

// Verify verifies the provided proof given the challenge, seed, and k.
func Verify(challenge string, seed []byte, k int, proof []uint64) error {
	f1, err := NewF1(k, seed)
	if err != nil {
		return err
	}

	var fxs []uint64
	for _, x := range proof {
		bucket, pos := findBucketAndPosForX(x)
		// TODO: Share calculations in case xs are found in the same bucket.
		fxBucket := f1.Calculate(bucket)
		fxs = append(fxs, bits.BytesToUint64(fxBucket[pos], k))
	}

	fx, err := NewFx(k, seed)
	if err != nil {
		return err
	}

	for t := 2; t <= 7; t++ {
		// fx.Calculate(t)
		for i := 0; i < int(math.Pow(float64(2), float64(7-t))); i++ {
			// 2 matches in the 1st table, 4 in the 2nd, 8 in the 3rd, and so on...
			step := int(math.Pow(float64(2), float64(t-1)))
			// TODO: Find whether fxs match; need to refactor WriteMatches to share code
			_ = step

			// Then calculate for the next table
			// fx.Calculate(t, fxs[])
			_ = fx
		}
	}

	return nil
}

// findBucketAndPosForX returns the first item of x's bucket and
// the position of x in the bucket.
func findBucketAndPosForX(x uint64) (uint64, int) {
	return x - (x % (8 * aes.BlockSize)), int(x) % (8 * aes.BlockSize)
}
