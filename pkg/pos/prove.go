package pos

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/spf13/afero"

	"github.com/kargakis/chiapos/pkg/serialize"
	"github.com/kargakis/chiapos/pkg/utils"
)

// Prove returns a space proof from the provided plot using the
// provided challenge.
func Prove(plotPath string, challenge []byte) ([]uint64, error) {
	fs := afero.NewOsFs()
	file, err := fs.Open(plotPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read plot: %w", err)
	}

	k, err := getK(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read k: %w", err)
	}
	fmt.Printf("Proving space with plot %s (k=%d)\n", plotPath, k)

	_, start, _, err := getLastTableIndexAndPositions(file)
	if err != nil {
		return nil, fmt.Errorf("cannot get last table indexes: %w", err)
	}

	entries, err := loadTable(file, start, k)
	if err != nil {
		return nil, fmt.Errorf("cannot load table into memory: %w", err)
	}

	challBig := new(big.Int).SetBytes(challenge)
	challBig = utils.Trunc(challBig, 0, k, challBig.BitLen())
	target := challBig.Uint64()

	index, err := getLastSmallerIndex(entries, target)
	if err != nil {
		return nil, fmt.Errorf("cannot get last index smaller than target %d: %w", target, err)
	}

	// Find all indices where f7 == target
	var read int
	var matches []*serialize.Entry
	entryLen := serialize.EntrySize(k, 7)
	for {
		entry, bytesRead, err := serialize.Read(file, int64(index+read), entryLen, k)
		if errors.Is(err, serialize.EOTErr) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read entry: %w", err)
		}
		read += bytesRead
		// TODO: Truncate fx to k bits
		if entry.Fx == target {
			matches = append(matches, entry)
		}
		// We are not going to find any more matches
		if entry.Fx > target {
			break
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no match found; no space proof exists for challenge %s", challenge)
	}

	return nil, nil
}

// getK returns k from the header of the provided plot.
func getK(file afero.File) (int, error) {
	kBytes := make([]byte, 1)
	if _, err := file.ReadAt(kBytes, 52); err != nil {
		return 0, err
	}
	return int(kBytes[0]), nil
}

func loadTable(file afero.File, start, k int) ([]*serialize.Entry, error) {
	var entries []*serialize.Entry
	var read int
	// Currently, the format of the C1 table is the same as the first table
	entryLen := serialize.EntrySize(k, 1)

	for {
		entry, bytesRead, err := serialize.Read(file, int64(start+read), entryLen, k)
		if errors.Is(err, serialize.EOTErr) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read entry: %w", err)
		}
		read += bytesRead
		entries = append(entries, entry)
	}

	return entries, nil
}

func getLastSmallerIndex(entries []*serialize.Entry, target uint64) (int, error) {
	var position int
	for _, e := range entries {
		if e.Fx < target {
			position = int(*e.X)
		} else {
			break
		}
	}
	return position, nil
}
