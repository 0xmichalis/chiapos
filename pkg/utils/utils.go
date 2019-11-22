package utils

import (
	"crypto/aes"
	"math/big"
	"math/bits"

	"github.com/kargakis/gochia/pkg/parameters"
)

// Concat performs zero-padded concatenation of the provided xs.
// Every member of xs is normalised to a [2^k] number.
// TODO: Maybe move normalisation out of here.
func Concat(k uint64, xs ...uint64) *big.Int {
	switch len(xs) {
	case 0:
		return big.NewInt(0)
	case 1:
		return new(big.Int).SetUint64(Normalise(xs[0], k))
	}
	res := big.NewInt(0)
	for _, x := range xs {
		x = Normalise(x, k)
		bigX := new(big.Int).SetUint64(x)
		res.Lsh(res, uint(k)).Add(res, bigX)
	}
	return res
}

// ConcatExtended shifts x paramEXT bits to the left, then adds
// y % paramM to it.
func ConcatExtended(x, y uint64) uint64 {
	tmp := x << parameters.ParamEXT
	tmp += y % parameters.ParamM
	return tmp
}

// Trunc returns the b most significant of x. If a is non-zero then the ath to (b-1)th
// bits of x are returned. x belongs to [2^k]
func Trunc(x *big.Int, a, b, k uint64) *big.Int {
	x.Rsh(x, uint(k-b))
	if a > 0 {
		least := big.NewInt(1)
		least.Lsh(least, uint(b-a))
		x.Mod(x, least)
	}
	return x
}

// IsAtMostKBits returns whether the provided number x is at
// most k bits.
func IsAtMostKBits(x, k uint64) bool {
	return k >= uint64(bits.Len64(x))
}

// Normalise normalises x if x is bigger than k bits
// by truncating x's least significant bits until x
// is k bits long.
func Normalise(x, k uint64) uint64 {
	if IsAtMostKBits(x, k) {
		return x
	}
	return x >> (uint64(bits.Len64(x)) - k)
}

// FillToBlock fills the provided byte slice with leading zeroes
// to match AES's block size requirements.
func FillToBlock(plain []byte) []byte {
	remainder := len(plain) % aes.BlockSize
	if remainder == 0 && len(plain) > 0 {
		return plain
	}

	leading := make([]byte, aes.BlockSize-remainder)
	return append(leading, plain...)
}
