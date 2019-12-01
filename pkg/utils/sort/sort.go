package sort

import (
	"bytes"
	"fmt"
	"io"
	"math/bits"
	"strconv"

	"github.com/spf13/afero"

	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

var buckets = make(map[string][]entry)

// bucketIndex returns the index of the target bucket for this
// entry. b is the smallest number such that 2^b >= 2 * num_entries.
func bucketIndex(entry uint64, b int) string {
	return string(mybits.Uint64ToBytes(entry)[:b-1])
}

// OnDisk performs sorting on the given file on disk, given begin which
// is the start of the data in the file in need of sorting, and availableMemory
// is the available memory in which sorting can be done.
func OnDisk(file, spare afero.File, begin, maxSize, availableMemory, entryLen, entryCount uint64) error {
	// TODO: FIXME - note that we need to take into account the
	// memory that will be used by loading the unsorted buckets,
	// the sorted buckets that are currently in memory, plus any
	// extra memory consumed by SortInMemory.
	if availableMemory > maxSize-begin {
		// if we can sort in memory, do that
		return InMemory(file, begin, entryLen, entryCount)
	}

	// The index in these buckets represents the common prefix
	// based on which we sort numbers (4 most-significant bits)
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

type entry struct {
	fx uint64
	x  uint64
}

func loadEntries(file afero.File, begin, entryLen, entryCount uint64) ([]entry, error) {
	tmpEntries := make([]byte, entryLen*entryCount)
	if _, err := file.ReadAt(tmpEntries, int64(begin)); err != nil {
		return nil, err
	}

	var entries []entry
	buf := bytes.NewBuffer(tmpEntries)
	for {
		line, err := buf.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		parts := bytes.Split(line, []byte(","))
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line read: %v", parts)
		}
		fx, _ := strconv.Atoi(string(parts[0]))
		x, _ := strconv.Atoi(string(parts[1]))
		entries = append(entries, entry{fx: uint64(fx), x: uint64(x)})
	}

	return entries, nil
}

// InMemory sorts the provided entries in memory.
func InMemory(file afero.File, begin, entryLen, entryCount uint64) error {
	entries, err := loadEntries(file, begin, entryLen, entryCount)
	if err != nil {
		return err
	}

	b := bits.Len64(uint64(2*len(entries))) / 8
	for _, e := range entries {
		bIndex := bucketIndex(e.fx, b)
		entries, ok := buckets[bIndex]
		if !ok {
			buckets[bIndex] = []entry{e}
		} else {
			index := -1
			for i, stored := range entries {
				if e.fx < stored.fx {
					index = i
					break
				}
			}
			if index != -1 {
				buckets[bIndex] = append(append(entries[:index], e), entries[index:]...)
			} else {
				buckets[bIndex] = append(buckets[bIndex], e)
			}
		}
	}
	return nil
}
