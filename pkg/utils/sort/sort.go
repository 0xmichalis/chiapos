package sort

import (
	"math/bits"
	"os"

	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

var buckets = make(map[string][]uint64)

// bucketIndex returns the index of the target bucket for this
// entry. b is the smallest number such that 2^b >= 2 * num_entries.
func bucketIndex(entry uint64, b int) string {
	return string(mybits.Uint64ToBytes(entry)[:b])
}

func SortOnDisk(file *os.File, availableMemory uint64) error {
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
func SortInMemory(entries []uint64) {
	bitsNum := bits.Len64(uint64(2 * len(entries)))
	b := 1 << bitsNum

	for _, entry := range entries {
		bIndex := bucketIndex(entry, b)
		entries, ok := buckets[bIndex]
		if !ok {
			buckets[bIndex] = []uint64{entry}
		} else {
			index := -1
			for i, stored := range entries {
				if entry < stored {
					index = i
					break
				}
			}
			if index != -1 {
				buckets[bIndex] = append(append(entries[:index], entry), entries[index:]...)
			} else {
				buckets[bIndex] = append(buckets[bIndex], entry)
			}
		}
	}
}
