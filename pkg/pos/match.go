package pos

import (
	"math"
	"math/big"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/serialize"
)

func init() {
	precomputeShifts()
}

// Precomputed shifts that specify which entries match with which other entries
// in adjacent buckets.
var matchingShifts = [2][parameters.ParamC]uint64{}

// Performs the precomputation of shifts.
func precomputeShifts() {
	for parity := 0; parity < 2; parity++ {
		for r := 0; r < parameters.ParamM; r++ {
			matchingShifts[parity][r] = uint64(math.Pow(float64(2*r+parity), 2)) % parameters.ParamC
		}
	}
}

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

var rightBids = [parameters.ParamC][]uint64{}
var rightPositions = [parameters.ParamC][]int{}

// WriteMatches compares the two buckets read from table t-1 and writes
// any matches in table t. The matching algorithm is carried over from
// the reference implementation since the naive approach is much slower.
func WriteMatches(file afero.File, fx *Fx, left, right []*serialize.Entry, currentStart, t, k int) (int, int, error) {
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

	var entries, wrote int
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
					leftID := le.Fx / parameters.ParamBC
					rightID := re.Fx / parameters.ParamBC
					if leftID+1 == rightID {
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

						f, err := fx.Calculate(t, le.Fx, leftMetadata, rightMetadata)
						if err != nil {
							return entries, wrote, err
						}
						// This is the collated output stored next to the entry - useful
						// for generating outputs for the next table.
						collated, err := Collate(t, uint64(k), leftMetadata, rightMetadata)
						if err != nil {
							return entries, wrote, err
						}
						// Now write the new output in the next table.
						index := uint64(le.Index)
						offset := uint64(re.Index - le.Index)
						w, err := serialize.Write(file, int64(currentStart+wrote), f, nil, &index, &offset, collated, k)
						if err != nil {
							return entries, wrote, err
						}
						entries++
						wrote += w
					}
				}
			}
		}
	}

	return entries, wrote, nil
}
