package pos

import (
	"math"

	"github.com/kargakis/gochia/pkg/parameters"
)

// Match is a matching function.
func Match(left, right uint64) bool {
	if parameters.BucketID(left)+1 != parameters.BucketID(right) {
		return false
	}
	bIDLeft, cIDLeft := parameters.GetIDs(left)
	bIDRight, cIDRight := parameters.GetIDs(right)

	//fmt.Println("left", left, "bid", bIDLeft, "cid", cIDLeft)
	//fmt.Println("right", right, "bid", bIDRight, "cid", cIDRight)

	for m := uint64(0); m < parameters.ParamM; m++ {
		firstCondition := (bIDRight-bIDLeft)%parameters.ParamB == m%parameters.ParamB
		secondCondition := (cIDRight-cIDLeft)%parameters.ParamC == uint64(math.Pow(float64(2*m+(parameters.BucketID(left)%2)), 2))%parameters.ParamC
		//fmt.Printf("1 %t: %d-%d (%d) == %d (%% %d)\n", firstCondition, bIDRight, bIDLeft, (bIDRight-bIDLeft)%parameters.ParamB, m, parameters.ParamB)
		//fmt.Printf("2 %t: %d-%d (%d) == %d (%% %d)\n", secondCondition, cIDRight, cIDLeft, (cIDRight-cIDLeft)%parameters.ParamC, uint64(math.Pow(float64(2*m+(parameters.BucketID(left)%2)), 2)), parameters.ParamC)
		if firstCondition && secondCondition {
			return true
		}
	}

	return false
}
