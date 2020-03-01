package sort

import (
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/serialize"
	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

var buckets = make(map[string][]*serialize.Entry)

// bucketIndex returns the index of the target bucket for this
// entry.
func bucketIndex(entry uint64, k int) string {
	// Keep the 8 most significant bits
	bitRepresentation := fmt.Sprintf("%08b", mybits.Uint64ToBytes(entry, k)[:1])
	// Drop [, ], and the 4 last significant bits
	return bitRepresentation[1 : len(bitRepresentation)/2]
}

// OnDisk performs sorting on the given file on disk, given begin which
// is the start of the data in the file in need of sorting, and availableMemory
// is the available memory in which sorting can be done.
func OnDisk(file afero.File, fs afero.Fs, begin, maxSize, availableMemory, entryLen, k, t int) error {
	// TODO: FIXME - note that we need to take into account the
	// memory that will be used by loading the unsorted buckets,
	// the sorted buckets that are currently in memory, plus any
	// extra memory consumed by SortInMemory.
	if availableMemory > maxSize-begin {
		// if we can sort in memory, do that
		fmt.Println("Sorting in memory...")
		return sortInMem(file, begin, entryLen, k, t)
	}
	fmt.Println("Sorting on disk...")

	// Sort plot into buckets
	var read, write int
	var exit bool
	for {
		// load an amount of entries that can fit into memory
		bucketIndexes, entriesBytes, err := sortInMemory(file, begin+read, entryLen, k)
		if errors.Is(err, serialize.EOTErr) {
			exit = true
		} else if err != nil {
			return err
		}
		read += entriesBytes

		for _, i := range bucketIndexes {
			spare, err := getFileForIndex(fs, i)
			if err != nil {
				return err
			}
			spareInfo, err := spare.Stat()
			if err != nil {
				return err
			}
			wrote, err := writeBuckets(spare, int(spareInfo.Size()), []string{i}, k)
			if err != nil {
				return err
			}
			write += wrote
		}

		if exit {
			break
		}
	}

	// At this point all buckets are sorted by the first 4 most significant bits
	// and we need to sort them even further, then write them back to the main
	// plot.
	// for _, bucket := range getBucketsInOrder() {
	// OnDisk(bucket, fs, 0)
	// }
	return nil
}

// inMemory sorts the provided entries in memory.
func inMemory(file afero.File, begin, entryLen int, k int) error {
	bucketIndexes, _, err := sortInMemory(file, begin, entryLen, k)
	if err != nil {
		return fmt.Errorf("failed to sort in memory: %w", err)
	}

	_, err = writeBuckets(file, begin, bucketIndexes, k)
	return err
}

// sortInMemory sorts in memory, then returns the sorted bucket indexes
// so callers can write the buckets on disk.
func sortInMemory(file afero.File, begin, entryLen int, k int) ([]string, int, error) {
	entries, read, err := loadEntries(file, begin, entryLen, k)
	if err != nil {
		return nil, read, fmt.Errorf("cannot load entries in memory: %w", err)
	}

	var bucketIndexes []string
	for _, e := range entries {
		bIndex := bucketIndex(e.Fx, k)
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
				buckets[bIndex] = append(append(bEntries[:index], e), bEntries[index:]...)
			} else {
				buckets[bIndex] = append(buckets[bIndex], e)
			}
		}
	}

	sort.Strings(bucketIndexes)
	return bucketIndexes, read, nil
}

func loadEntries(file afero.File, begin, entryLen, k int) (entries []*serialize.Entry, read int, err error) {
	for {
		entry, readLen, err := serialize.Read(file, int64(begin+read), entryLen, k)
		if errors.Is(err, serialize.EOTErr) || errors.Is(err, io.EOF) {
			return entries, read + readLen, nil
		}
		if err != nil {
			return entries, read + readLen, err
		}
		read += readLen
		entries = append(entries, entry)
	}
}

// writeBuckets writes the buckets for the provided indexes in file.
func writeBuckets(file afero.File, begin int, bucketIndexes []string, k int) (int, error) {
	var wrote int

	for _, index := range bucketIndexes {
		for _, e := range buckets[index] {
			n, err := serialize.Write(file, int64(begin+wrote), e.Fx, e.X, e.Pos, e.Offset, e.Collated, k)
			if err != nil {
				return wrote + n, fmt.Errorf("cannot write sorted values: %w", err)
			}
			wrote += n
		}
	}

	return wrote, nil
}

// a cache of all the files backing the buckets
var bucketStore = make(map[string]afero.File)

func getFileForIndex(fs afero.Fs, i string) (afero.File, error) {
	f, exists := bucketStore[i]
	if !exists {
		var err error
		f, err = fs.Create("bucket-" + i)
		if err != nil {
			return nil, err
		}
		bucketStore[i] = f
	}
	return f, nil
}

func getBucketsInOrder() []afero.File {
	var bIndexes []string
	for i := range bucketStore {
		bIndexes = append(bIndexes, i)
	}
	sort.Strings(bIndexes)

	var buckets []afero.File
	for _, i := range bIndexes {
		buckets = append(buckets, bucketStore[i])
	}

	return buckets
}

// sortInMem sorts a table in memory.
func sortInMem(file afero.File, begin, entryLen int, k, t int) error {
	entries, _, err := loadEntries(file, begin, entryLen, k)
	if err != nil {
		return fmt.Errorf("cannot load entries in memory: %w", err)
	}

	sort.Sort(serialize.ByOutput{Entries: entries, TableIndex: t})

	var wrote int
	for _, e := range entries {
		n, err := serialize.Write(file, int64(begin+wrote), e.Fx, e.X, e.Pos, e.Offset, e.Collated, k)
		if err != nil {
			return fmt.Errorf("cannot write sorted values: %w", err)
		}
		wrote += n
	}

	return nil
}
