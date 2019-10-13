package utils

import "math/big"

// Set returns the set of x -> [X] is the set {0, 1, ..., X-1}
func Set(x uint) []uint {
	set := make([]uint, x)
	for i := uint(0); i < x; i++ {
		set[i] = i
	}
	return set
}

// Concat performs zero-padded concatenation of x and y
// y belongs to [2^k]
func Concat(x, y *big.Int, k uint64) *big.Int {
	return x.Lsh(x, uint(k)).Add(x, y)
}

// Trunc returns the b most significant of x. If a is non-zero then the ath to (b-1)th
// bits of x are returned. x belongs to [2^k]
func Trunc(x *big.Int, a, b, k uint64) *big.Int {
	x.Rsh(x, uint(k-b))
	if a > 0 {
		least := big.NewInt(1)
		least.Lsh(least, uint(a))
		x.Mod(x, least)
	}
	return x
}
