package bits

import "encoding/binary"

// Uint64ToBytes converts an unsigned 64-bit integer
// to a byte slice. The returned order used is big endian,
// similar to the big.Int api.
func Uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// BytesToUint64 converts a byte slice to an unsigned
// 64-bit integer. The provided byte slice is expected
// to be of size 8. The expected order used in the input
// slice is big endian, similar to the big.Int api.
func BytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}
