package pos

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/serialize"
	"github.com/kargakis/chiapos/pkg/utils"
)

// Checkpoint reads the last table in the plot and creates a new
// table where it stores checkpoints to the last table so fast
// retrieval of proofs can be enabled by reading the checkpoints.
// TODO: Create checkpoint table C2 to checkpoint C1.
func Checkpoint(file afero.File, k int) (int, error) {
	fmt.Println("Starting checkpointing...")
	var wrote int

	_, start, end, err := getLastTableIndexAndPositions(file)
	if err != nil {
		return wrote, err
	}

	var bytesRead, read, count int
	var entry *serialize.Entry
	entryLen := serialize.EntrySize(k, 7)
	for {
		// Create checkpoints of the last table every C1 entries
		entry, bytesRead, err = serialize.Read(file, int64(start+read), entryLen, k)
		if errors.Is(err, serialize.EOTErr) || errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return wrote, fmt.Errorf("cannot read left entry: %w", err)
		}

		if count%parameters.ParamC1 == 0 {
			// Write down the exact position of the checkpointed entry in the plot.
			pos := uint64(start + read)
			w, err := serialize.Write(file, int64(end+1+wrote), entry.Fx, nil, &pos, nil, nil, k)
			if err != nil {
				return wrote + w, err
			}
			wrote += w
		}
		count++
		read += bytesRead
	}

	eotBytes, err := WriteEOT(file, entryLen)
	if err != nil {
		return wrote + eotBytes, err
	}
	wrote += eotBytes

	// TODO: Set a different index than 8, change index to a string
	if err := updateLastTableIndexAndPositions(file, 8, end+1, end+1+wrote); err != nil {
		return wrote, err
	}
	fmt.Printf("Finished checkpointing (wrote %s)\n", utils.PrettySize(float64(wrote)))

	return wrote, nil
}
