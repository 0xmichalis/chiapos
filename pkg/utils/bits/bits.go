package bits

import "encoding/binary"

// Uint64ToBytes converts an unsigned 64-bit integer
// to a byte slice.
func Uint64ToBytes(n uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, n)
	return b
}

// BytesToUint64 converts a byte slice to an unsigned
// 64-bit integer. The provided byte slice is expected
// to be of size 8.
func BytesToUint64(b []byte) uint64 {
	return binary.LittleEndian.Uint64(b)
}
