package bits

import (
	"math/bits"
)

// bigEndian is a variable-size bits implementation supporting
// up to 64bits integers. Serialization to bytes is done
// using big-endian order.
type bigEndian struct {
	// Len tracks how many bits are stored
	Len int
}

func (be bigEndian) Uint64(b []byte) uint64 {
	bitLen := be.Len

	switch {
	case bitLen <= 8:
		_ = b[0] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[0])
	case bitLen > 8 && bitLen <= 16:
		_ = b[1] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[1]) | uint64(b[0])<<8
	case bitLen > 16 && bitLen <= 24:
		_ = b[2] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[2]) | uint64(b[1])<<8 | uint64(b[0])<<16
	case bitLen > 24 && bitLen <= 32:
		_ = b[3] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[3]) | uint64(b[2])<<8 | uint64(b[1])<<16 | uint64(b[0])<<24
	case bitLen > 32 && bitLen <= 40:
		_ = b[4] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[4]) | uint64(b[3])<<8 | uint64(b[2])<<16 | uint64(b[1])<<24 |
			uint64(b[0])<<32
	case bitLen > 40 && bitLen <= 48:
		_ = b[5] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[5]) | uint64(b[4])<<8 | uint64(b[3])<<16 | uint64(b[2])<<24 |
			uint64(b[1])<<32 | uint64(b[0])<<40
	case bitLen > 48 && bitLen <= 56:
		_ = b[6] // bounds check hint to compiler; see golang.org/issue/14808
		return uint64(b[6]) | uint64(b[5])<<8 | uint64(b[4])<<16 | uint64(b[3])<<24 |
			uint64(b[2])<<32 | uint64(b[1])<<40 | uint64(b[0])<<48
	}
	_ = b[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
}

func (be bigEndian) PutUint64(b []byte, v uint64) {
	bitLen := be.Len

	switch {
	case bitLen <= 8:
		_ = b[0] // early bounds check to guarantee safety of writes below
		b[0] = byte(v)
	case bitLen > 8 && bitLen <= 16:
		_ = b[1] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 8)
		b[1] = byte(v)
	case bitLen > 16 && bitLen <= 24:
		_ = b[2] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 16)
		b[1] = byte(v >> 8)
		b[2] = byte(v)
	case bitLen > 24 && bitLen <= 32:
		_ = b[3] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 24)
		b[1] = byte(v >> 16)
		b[2] = byte(v >> 8)
		b[3] = byte(v)
	case bitLen > 32 && bitLen <= 40:
		_ = b[4] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 32)
		b[1] = byte(v >> 24)
		b[2] = byte(v >> 16)
		b[3] = byte(v >> 8)
		b[4] = byte(v)
	case bitLen > 40 && bitLen <= 48:
		_ = b[5] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 40)
		b[1] = byte(v >> 32)
		b[2] = byte(v >> 24)
		b[3] = byte(v >> 16)
		b[4] = byte(v >> 8)
		b[5] = byte(v)
	case bitLen > 48 && bitLen <= 56:
		_ = b[6] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 48)
		b[1] = byte(v >> 40)
		b[2] = byte(v >> 32)
		b[3] = byte(v >> 24)
		b[4] = byte(v >> 16)
		b[5] = byte(v >> 8)
		b[6] = byte(v)
	case bitLen > 56:
		_ = b[7] // early bounds check to guarantee safety of writes below
		b[0] = byte(v >> 56)
		b[1] = byte(v >> 48)
		b[2] = byte(v >> 40)
		b[3] = byte(v >> 32)
		b[4] = byte(v >> 24)
		b[5] = byte(v >> 16)
		b[6] = byte(v >> 8)
		b[7] = byte(v)
	}
}

// ToBytes returns the minimum bytes necessary
// to store a k-bit number.
func ToBytes(k int) int {
	bSize := k / 8
	if k%8 != 0 {
		bSize++
	}
	return bSize
}

// Uint64ToBytes converts an unsigned 64-bit integer
// to a byte slice. The returned order used is big endian,
// similar to the big.Int api. Even though a 64-bits integer
// is provided, only enough bytes necessary to represent the
// integer are serialized.
func Uint64ToBytes(n uint64, k int) []byte {
	b := make([]byte, ToBytes(k))
	bigEndian{Len: k}.PutUint64(b, n)
	return b
}

// BytesToUint64 converts a byte slice to an unsigned
// 64-bit integer. The expected order used in the input
// slice is big endian, similar to the big.Int api.
func BytesToUint64(b []byte, k int) uint64 {
	return bigEndian{Len: k}.Uint64(b)
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
