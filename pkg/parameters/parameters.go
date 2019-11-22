package parameters

const (
	ParamEXT = 5

	ParamM = 1 << ParamEXT

	ParamB = 60

	ParamC = 509

	ParamBC = ParamB * ParamC

	ParamC1 = 10000
	ParamC2 = 10000

	ParamStubBits = 4
)

func BucketID(x uint64) uint64 {
	return x / ParamBC
}

func GetIDs(x uint64) (bID, cID uint64) {
	y := x % ParamBC
	return y / ParamC, y % ParamC
}
