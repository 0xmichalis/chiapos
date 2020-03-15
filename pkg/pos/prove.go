package pos

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/bits"

	"github.com/spf13/afero"

	"github.com/kargakis/chiapos/pkg/serialize"
	"github.com/kargakis/chiapos/pkg/utils"
	bitsutil "github.com/kargakis/chiapos/pkg/utils/bits"
	fsutil "github.com/kargakis/chiapos/pkg/utils/fs"
)

type SpaceProof []uint64

func (sp SpaceProof) String() string {
	var proofString string
	for i, p := range sp {
		proofString += fmt.Sprintf("%d", p)
		if i != len(sp)-1 {
			proofString += ","
		}
	}
	return proofString
}

// Prove returns a space proof from the provided plot using the
// provided challenge.
func Prove(plotPath, fsType string, challenge []byte) (SpaceProof, error) {
	fs, err := fsutil.GetFs(fsType)
	if err != nil {
		return nil, err
	}
	file, err := fs.Open(plotPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read plot: %w", err)
	}
	defer file.Close()

	k, err := getK(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read k: %w", err)
	}
	// fmt.Printf("Proving space with plot %s (k=%d)\n", plotPath, k)

	// get C1 start index
	_, start, _, err := getLastTableIndexAndPositions(file)
	if err != nil {
		return nil, fmt.Errorf("cannot get last table indexes: %w", err)
	}

	// load C1 in memory
	// fmt.Println("Loading C1 table in memory...")
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
	// fmt.Println("Searching for f7 outputs matching the challenge...",)
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
		// Truncate fx to k bits
		fEntry := utils.TruncPrimitive(entry.Fx, 0, k, bits.Len64(entry.Fx))
		if fEntry == target {
			matches = append(matches, entry)
		}
		// We are not going to find any more matches
		if fEntry > target {
			break
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no match found; no space proof exists for challenge %d", target)
	}

	// TODO: Make this index configurable
	matchIndex := 0
	leftPos := *matches[matchIndex].Pos
	rightPos := *matches[matchIndex].Pos + *matches[matchIndex].Offset

	proof, err := getInputs(file, 6, k, leftPos, rightPos)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve proof from plot: %w", err)
	}
	if len(proof) != 64 {
		return nil, fmt.Errorf("invalid proof: expected 64 x values, got %d", len(proof))
	}
	return proof, nil
}

// getK returns k from the header of the provided plot.
func getK(file afero.File) (int, error) {
	kBytes := make([]byte, 1)
	if _, err := file.ReadAt(kBytes, int64(len(plotHeader)+utils.KeyLen)); err != nil {
		return 0, err
	}
	return int(bitsutil.BytesToUint64(kBytes, 1)), nil
}

func loadTable(file afero.File, start, k int) ([]*serialize.Entry, error) {
	var entries []*serialize.Entry

	if _, err := file.Seek(int64(start), io.SeekStart); err != nil {
		return nil, err
	}
	buf := bufio.NewReader(file)

	for {
		entry, err := serialize.ReadCheckpoint(buf, k)
		if errors.Is(err, serialize.EOTErr) || errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func getLastSmallerIndex(entries []*serialize.Entry, target uint64) (int, error) {
	var position int
	for _, e := range entries {
		if e.Fx < target {
			position = int(*e.Pos)
		} else {
			break
		}
	}
	if position == 0 {
		return 0, fmt.Errorf("no position found")
	}
	return position, nil
}

// getInputs walks all tables recursively until it reaches the last table
// to retrieve all the 64 x values comprising a proof of space.
func getInputs(file afero.File, t, k int, leftPos, rightPos uint64) ([]uint64, error) {
	entryLen := serialize.EntrySize(k, t)
	leftEntry, _, err := serialize.Read(file, int64(leftPos), entryLen, k)
	if err != nil {
		return nil, fmt.Errorf("cannot read left entry at table %d: %w", t, err)
	}
	rightEntry, _, err := serialize.Read(file, int64(rightPos), entryLen, k)
	if err != nil {
		return nil, fmt.Errorf("cannot read right entry at table %d: %w", t, err)
	}

	if t == 1 {
		return []uint64{*leftEntry.X, *rightEntry.X}, nil
	}

	// aggregate inputs from previous table and forward to the next
	left, err := getInputs(file, t-1, k, *leftEntry.Pos, *leftEntry.Pos+*leftEntry.Offset)
	if err != nil {
		return nil, fmt.Errorf("cannot get inputs for left entry at table %d: %w", t, err)
	}
	right, err := getInputs(file, t-1, k, *rightEntry.Pos, *rightEntry.Pos+*rightEntry.Offset)
	if err != nil {
		return nil, fmt.Errorf("cannot get inputs for right entry at table %d: %w", t, err)
	}
	return append(left, right...), nil
}
