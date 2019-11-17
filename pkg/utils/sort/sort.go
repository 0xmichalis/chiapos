package sort

import (
	"math/bits"
	"os"
)

func SortOnDisk(file *os.File) error {
	bucketSizes := make([]int, 16)
	bucketStarts := make([]int, 16)

	_ = bucketSizes
	_ = bucketStarts

	var total int
	_ = total
	for i := 0; i < 16; i++ {

	}
	return nil
}

// SortInMemory sorts the provided entries in memory.
func SortInMemory(entries []uint64) []uint64 {
	bitsNum := bits.Len64(uint64(2 * len(entries)))
	b := 1 << bitsNum
	_ = b

	for _, entry := range entries {
		_ = entry
	}

	return nil
}
