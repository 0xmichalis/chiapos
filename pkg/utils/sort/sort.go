package sort

import (
	"fmt"
	"math/bits"
	"sort"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/serialize"
	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

var buckets = make(map[string][]*serialize.Entry)

// bucketIndex returns the index of the target bucket for this
// entry. b is the smallest number such that 2^b >= 2 * num_entries.
func bucketIndex(entry uint64, b, k int) string {
	return string(mybits.Uint64ToBytes(entry, k)[:b-1])
}

// OnDisk performs sorting on the given file on disk, given begin which
// is the start of the data in the file in need of sorting, and availableMemory
// is the available memory in which sorting can be done.
func OnDisk(file, spare afero.File, begin, maxSize, availableMemory, entryLen, entryCount, k int) error {
	// TODO: FIXME - note that we need to take into account the
	// memory that will be used by loading the unsorted buckets,
	// the sorted buckets that are currently in memory, plus any
	// extra memory consumed by SortInMemory.
	if availableMemory > maxSize-begin {
		// if we can sort in memory, do that
		return InMemory(file, begin, entryLen, entryCount, k)
	}

	// The index in these buckets represents the common prefix
	// based on which we sort numbers (4 most-significant bits)
	//bucketSizes := make([]int, 16)
	//bucketBegins := make([]int, 16)
	//filePositions := make([]int, 16)

	var read int
	for i := 0; i < entryCount; i++ {
		_, readLen, err := serialize.Read(file, int64(begin+read), entryLen, k)
		if err != nil {
			return err
		}
		read += readLen

		// read until we reach 2/3 of available memory
	}
	return nil
}

func loadEntries(file afero.File, begin, entryLen, entryCount, k int) (entries []*serialize.Entry, read int, err error) {
	for i := 0; i < entryCount; i++ {
		entry, readLen, err := serialize.Read(file, int64(begin+read), entryLen, k)
		if err != nil {
			return nil, read + readLen, err
		}
		read += readLen
		entries = append(entries, entry)
	}

	return entries, read, nil
}

// InMemory sorts the provided entries in memory.
func InMemory(file afero.File, begin, entryLen, entryCount int, k int) error {
	bucketIndexes, err := sortInMemory(file, begin, entryLen, entryCount, k)
	if err != nil {
		return fmt.Errorf("cannot load entries in memory: %v", err)
	}

	_, err = WriteBuckets(file, begin, bucketIndexes, k)
	return err
}

// sortInMemory sorts in memory, then returns the sorted bucket indexes
// so callers can write the buckets on disk.
func sortInMemory(file afero.File, begin, entryLen, entryCount int, k int) ([]string, error) {
	entries, _, err := loadEntries(file, begin, entryLen, entryCount, k)
	if err != nil {
		return nil, fmt.Errorf("cannot load entries in memory: %v", err)
	}

	var bucketIndexes []string
	// TODO: Handle case where entries is small
	b := bits.Len64(uint64(2*len(entries))) / 8
	for _, e := range entries {
		bIndex := bucketIndex(e.Fx, b, k)
		bEntries, ok := buckets[bIndex]
		if !ok {
			buckets[bIndex] = []*serialize.Entry{e}
			bucketIndexes = append(bucketIndexes, bIndex)
		} else {
			index := -1
			for i, stored := range bEntries {
				if e.Fx < stored.Fx {
					index = i
					break
				}
			}
			if index != -1 {
				buckets[bIndex] = append(append(bEntries[:index], e), bEntries[index+1:]...)
			} else {
				buckets[bIndex] = append(buckets[bIndex], e)
			}
		}
	}

	sort.Strings(bucketIndexes)
	return bucketIndexes, nil
}

func WriteBuckets(file afero.File, begin int, bucketIndexes []string, k int) (int, error) {
	var wrote int

	for _, index := range bucketIndexes {
		for _, e := range buckets[index] {
			n, err := serialize.Write(file, int64(begin+wrote), e.Fx, e.X, e.Pos, e.Offset, e.Collated, k)
			if err != nil {
				return wrote + n, fmt.Errorf("cannot write sorted values: %v", err)
			}
			wrote += n
		}
	}

	return wrote, nil
}
