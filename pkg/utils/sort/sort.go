package sort

import (
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/afero"

	"github.com/kargakis/chiapos/pkg/serialize"
)

// OnDisk performs sorting on the given file on disk, given begin which
// is the start of the data in the file in need of sorting, and availableMemory
// is the available memory in which sorting can be done.
func OnDisk(file afero.File, fs afero.Fs, begin, tableSize, availableMemory, k, t int) error {
	entryLen := serialize.EntrySize(k, t)

	// TODO: Implement sort on disk (https://github.com/kargakis/chiapos/issues/5)
	return sortInMemory(file, begin, entryLen, k, t)
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

// sortInMemory sorts a table in memory.
func sortInMemory(file afero.File, begin, entryLen int, k, t int) error {
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
