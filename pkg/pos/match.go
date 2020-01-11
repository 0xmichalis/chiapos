package pos

import (
	"math"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/serialize"
)

// FindMatches compares the two buckets and returns any matches.
func FindMatches(left, right []*serialize.Entry) map[uint64]uint64 {
	matches := make(map[uint64]uint64)

	for _, le := range left {
		for _, re := range right {
			if Match(le.Fx, re.Fx) {
				matches[le.Fx] = re.Fx
			}
		}
	}

	return matches
}

// Match is a matching function.
func Match(left, right uint64) bool {
	if parameters.BucketID(left)+1 != parameters.BucketID(right) {
		return false
	}
	bIDLeft, cIDLeft := parameters.GetIDs(left)
	bIDRight, cIDRight := parameters.GetIDs(right)

	for m := uint64(0); m < parameters.ParamM; m++ {
		firstCondition := (bIDRight-bIDLeft)%parameters.ParamB == m%parameters.ParamB
		secondCondition := (cIDRight-cIDLeft)%parameters.ParamC == uint64(math.Pow(float64(2*m+(parameters.BucketID(left)%2)), 2))%parameters.ParamC
		if firstCondition && secondCondition {
			return true
		}
	}

	return false
}
