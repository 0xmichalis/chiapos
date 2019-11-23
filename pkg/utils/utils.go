package utils

import (
	"crypto/aes"
	"fmt"
	"math/big"
	"runtime"

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
		return new(big.Int).SetUint64(xs[0])
	}
	res := big.NewInt(0)
	for _, x := range xs {
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

// PrintMemUsage outputs the current, total and OS memory being used, as well
// as the number of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
