package parameters

const (
	// ParamEXT defines the additional bits to be added to any function
	// output to reduce the impact of collisions on the matching function.
	ParamEXT = 5
	ParamM   = 1 << ParamEXT

	// B and C groups which constitute a bucket, or BC group. These groups determine how
	// elements match with each other. Two elements must be in adjacent buckets to match.
	ParamB  = 60
	ParamC  = 509
	ParamBC = ParamB * ParamC

	// ParamC1 defines how many entries to checkpoint from the last table
	// to enable fast lookups.
	ParamC1 = 10000

	// Space parameters controlling the plot size.

	// Must be set high enough to prevent attacks of fast plotting
	KMinPlotSize = 16
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
