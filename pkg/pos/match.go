package pos

import (
	"math"
	"math/big"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/serialize"
)

type Match struct {
	Left  uint64
	Right uint64

	LeftPosition uint64
	// Offset is used to estimate the position of the right match
	// in the table, which is LeftPosition + Offset.
	Offset uint64

	LeftMetadata  *big.Int
	RightMetadata *big.Int
}

// FindMatches compares the two buckets and returns any matches.
func FindMatches(left, right []*serialize.Entry) []Match {
	var matches []Match
	for _, le := range left {
		for _, re := range right {
			if matchEntries(le.Fx, re.Fx) {
				var leftMetadata, rightMetadata *big.Int
				if le.X != nil {
					leftMetadata = big.NewInt(int64(*le.X))
				} else if le.Collated != nil {
					leftMetadata = le.Collated
				}
				if re.X != nil {
					rightMetadata = big.NewInt(int64(*re.X))
				} else if re.Collated != nil {
					rightMetadata = re.Collated
				}
				matches = append(matches, Match{
					Left:          le.Fx,
					Right:         re.Fx,
					LeftPosition:  uint64(le.Index),
					Offset:        uint64(re.Index - le.Index),
					LeftMetadata:  leftMetadata,
					RightMetadata: rightMetadata,
				})
			}
		}
	}

	return matches
}

// matchEntries is a matching function.
func matchEntries(left, right uint64) bool {
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
