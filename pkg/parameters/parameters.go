package parameters

const (
	ParamEXT = 5

	ParamM = 1 << ParamEXT

	// B and C groups which constitute a bucket, or BC group. These groups determine how
	// elements match with each other. Two elements must be in adjacent buckets to match.
	ParamB = 60
	ParamC = 509

	ParamBC = ParamB * ParamC

	// Must be set high enough to prevent attacks of fast plotting
	// TODO: Should be set to 33
	KMinPlotSize = 15

	// Set at 59 to allow easy use of 64 bit integers
	KMaxPlotSize = 59
)

func BucketID(x uint64) uint64 {
	return x / ParamBC
}

func GetIDs(x uint64) (bID, cID uint64) {
	y := x % ParamBC
	return y / ParamC, y % ParamC
}
