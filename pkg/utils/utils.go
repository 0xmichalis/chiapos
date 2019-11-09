package utils

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"math/big"
	"math/bits"
)

// Set returns the set of x -> [X] is the set {0, 1, ..., X-1}
func Set(x uint) []uint {
	set := make([]uint, x)
	for i := uint(0); i < x; i++ {
		set[i] = i
	}
	return set
}

// Concat performs zero-padded concatenation of the provided xs.
// Every member of xs is normalised to a [2^k] number.
// TODO: Maybe move normalisation out of here.
func Concat(k uint64, xs ...uint64) *big.Int {
	switch len(xs) {
	case 0:
		return big.NewInt(0)
	case 1:
		return big.NewInt(0).SetUint64(Normalise(xs[0], k))
	}
	res := big.NewInt(0)
	for _, x := range xs {
		x = Normalise(x, k)
		bigX := big.NewInt(0).SetUint64(x)
		res.Lsh(res, uint(k)).Add(res, bigX)
	}
	return res
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

// CollaSize returns the collation size for t.
func CollaSize(t int) (size *int, err error) {
	size = new(int)
	switch t {
	case 2:
		*size = 1
	case 3, 7:
		*size = 2
	case 4, 5:
		*size = 4
	case 6:
		*size = 3
	default:
		return nil, fmt.Errorf("collation size for t=%d is undefined", t)
	}
	return
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

// Ct is a collation function for t.
func Ct(t int, k uint64, x ...uint64) (*big.Int, error) {
	if t < 2 || t > 7 {
		return nil, fmt.Errorf("collation function for t=%d is undefined", t)
	}
	if len(x) != 2^(t-2) {
		return nil, fmt.Errorf("invalid x count: %d, expected %d", len(x), 2^(t-2))
	}

	switch t {
	case 2:
		return big.NewInt(0).SetUint64(x[0]), nil

	case 3:
		return Concat(k, x[0], x[1]), nil

	case 4:
		return Concat(k, x[0], x[1], x[2], x[3]), nil

	case 5:
		left := Concat(k, x[0], x[1], x[2], x[3])
		right := Concat(k, x[4], x[5], x[6], x[7])
		return left.Xor(left, right), nil

	case 6:
		first := Concat(k, x[0], x[1], x[2])
		second := Concat(k, x[4], x[5], x[6])
		third := Concat(k, x[8], x[9], x[10])
		fourth := Concat(k, x[12], x[13], x[14])
		return first.Xor(first, second).Xor(first, third).Xor(first, fourth), nil

	case 7:
		first := Concat(k, x[0], x[1])
		second := Concat(k, x[4], x[5])
		third := Concat(k, x[8], x[9])
		fourth := Concat(k, x[12], x[13])
		fifth := Concat(k, x[16], x[17])
		sixth := Concat(k, x[20], x[21])
		seventh := Concat(k, x[24], x[25])
		eighth := Concat(k, x[28], x[29])
		return first.Xor(first, second).
			Xor(first, third).
			Xor(first, fourth).
			Xor(first, fifth).
			Xor(first, sixth).
			Xor(first, seventh).
			Xor(first, eighth), nil
	}
	return nil, errors.New("should never reach here")
}

// At is a high-level hash function that calls AES on its inputs.
// c is meant to be created using the plot seed as a key.
func At(x, y, k uint64, t int, c cipher.Block) (*uint64, error) {
	param := big.NewInt(1).Rsh(big.NewInt(1), 128)

	// setup x low and high
	xBig := big.NewInt(0).SetUint64(x)
	xLow := big.NewInt(0)
	xHigh := big.NewInt(0)
	xHigh.DivMod(xBig, param, xLow)

	// setup y low and high
	yBig := big.NewInt(0).SetUint64(y)
	yLow := big.NewInt(0)
	yHigh := big.NewInt(0)
	yHigh.DivMod(yBig, param, yLow)

	// setup size
	collaSize, err := CollaSize(t)
	if err != nil {
		return nil, err
	}
	size := 2 * int(k) * *collaSize

	// main logic
	var cipherText []byte
	switch {
	case 0 <= size && size <= 128:
		plaintext := Concat(k, x, y)
		c.Encrypt(cipherText, plaintext.Bytes())

	case 129 <= size && size <= 256:
		c.Encrypt(cipherText, xBig.Bytes())
		tmp := big.NewInt(0).SetBytes(cipherText)
		c.Encrypt(cipherText, tmp.Xor(tmp, yBig).Bytes())

	case 257 <= size && size <= 384:
		var cipherConcat []byte
		c.Encrypt(cipherConcat, Concat(k, xLow.Uint64(), yLow.Uint64()).Bytes())
		ccBig := big.NewInt(0).SetBytes(cipherConcat)

		var cipherYHigh []byte
		c.Encrypt(cipherYHigh, yHigh.Bytes())
		cyBig := big.NewInt(0).SetBytes(cipherYHigh)

		var cipherXHigh []byte
		c.Encrypt(cipherXHigh, xHigh.Bytes())
		cxBig := big.NewInt(0).SetBytes(cipherXHigh)

		ccBig.Xor(ccBig, cyBig).Xor(ccBig, cxBig)
		c.Encrypt(cipherText, ccBig.Bytes())

	case 385 <= size && size <= 512:
		var tmp []byte
		c.Encrypt(tmp, xHigh.Bytes())
		tmpBig := big.NewInt(0).SetBytes(tmp)
		c.Encrypt(tmp, tmpBig.Xor(tmpBig, xLow).Bytes())
		tmpBig = big.NewInt(0).SetBytes(tmp)

		var cipherYHigh []byte
		c.Encrypt(cipherYHigh, yHigh.Bytes())
		cyBig := big.NewInt(0).SetBytes(cipherYHigh)

		c.Encrypt(cipherText, tmpBig.Xor(tmpBig, cyBig).Xor(tmpBig, yLow).Bytes())
	}

	// need to return the most significant k+paramEXT bits
	res := big.NewInt(0).SetBytes(cipherText)
	r := Trunc(res, 0, k+5, k).Uint64()
	return &r, nil
}
