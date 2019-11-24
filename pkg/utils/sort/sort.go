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

// SortOnDisk performs sorting on the given file on disk, given begin which
// is the start of the data in the file in need of sorting, and availableMemory
// is the available memory in which sorting can be done.
func SortOnDisk(file *os.File, begin, maxSize, availableMemory, entryLen uint64) error {
	// TODO: FIXME - note that we need to take into account the
	// memory that will be used by loading the unsorted buckets,
	// the sorted buckets that are currently in memory, plus any
	// extra memory consumed by SortInMemory.
	if availableMemory > maxSize-begin {
		// if we can sort in memory, do that
		SortInMemory(nil)
		return nil
	}

	bucketSizes := make([]uint64, 16)
	bucketBegins := make([]uint64, 16)
	filePositions := make([]uint64, 16)

	var total uint64
	for i := 0; i < 16; i++ {
		bucketBegins[i] = total
		total += bucketSizes[i]
		filePositions[i] = begin + bucketBegins[i]*entryLen
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
