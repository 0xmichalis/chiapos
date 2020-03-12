package pos

import (
	"math"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/serialize"
)

func init() {
	precomputeShifts()
}

// Precomputed shifts that specify which entries match with which other entries
// in adjacent buckets.
var matchingShifts = [2][parameters.ParamM]uint64{}

// Performs the precomputation of shifts.
func precomputeShifts() {
	for parity := 0; parity < 2; parity++ {
		for r := 0; r < parameters.ParamM; r++ {
			matchingShifts[parity][r] = uint64(math.Pow(float64(2*r+parity), 2)) % parameters.ParamC
		}
	}
}

type Match struct {
	Left  *serialize.Entry
	Right *serialize.Entry
}

var rightBids = [parameters.ParamC][]uint64{}
var rightPositions = [parameters.ParamC][]int{}

// FindMatches compares the two buckets read from table t-1 and returns
// any matches. The matching algorithm is carried over from the reference
// implementation since the naive approach is much slower.
func FindMatches(left, right []*serialize.Entry) []*Match {
	for i := 0; i < parameters.ParamC; i++ {
		rightBids[i] = nil
		rightPositions[i] = nil
	}

	parity := (left[0].Fx / parameters.ParamBC) % 2

	for i := range right {
		rightFx := right[i].Fx % parameters.ParamC
		rightBids[rightFx] = append(rightBids[rightFx], (rightFx%parameters.ParamBC)/parameters.ParamC)
		rightPositions[rightFx] = append(rightPositions[rightFx], i)
	}

	var matches []*Match
	for leftIndex := range left {
		leftBid := (left[leftIndex].Fx % parameters.ParamBC) / parameters.ParamC
		leftCid := left[leftIndex].Fx % parameters.ParamC

		for m := uint64(0); m < parameters.ParamM; m++ {
			targetBid := leftBid + m
			targetCid := leftCid + matchingShifts[parity][m]

			// This is faster than %
			if targetBid >= parameters.ParamB {
				targetBid -= parameters.ParamB
			}
			if targetCid >= parameters.ParamC {
				targetCid -= parameters.ParamC
			}

			for i := 0; i < len(rightBids[targetCid]); i++ {
				rightBid := rightBids[targetCid][i]
				if targetBid == rightBid {
					le := left[leftIndex]
					re := right[rightPositions[targetCid][i]]
					leftID := parameters.BucketID(le.Fx)
					rightID := parameters.BucketID(re.Fx)
					if leftID+1 == rightID {
						matches = append(matches, &Match{Left: le, Right: re})
					}
				}
			}
		}
	}

	return matches
}

// matchNaive is the naive implementation of the matching function.
func matchNaive(left, right uint64) bool {
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
