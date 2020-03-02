package pos

import (
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/serialize"
	"github.com/kargakis/gochia/pkg/utils"
	"github.com/kargakis/gochia/pkg/utils/bits"
	"github.com/kargakis/gochia/pkg/utils/sort"
)

// This is Phase 1, or forward propagation. During this phase, all of the 7 tables,
// and f functions, are evaluated. The result is an intermediate plot file, that is
// several times larger than what the final file will be, but that has all of the
// proofs of space in it. First, F1 is computed, which is special since it uses
// AES256, and each encryption provides multiple output values. Then, the rest of the
// f functions are computed, and a sort on disk happens for each table.
func WritePlotFile(filename string, k, availableMemory int, id []byte, retry bool) error {
	fs := afero.NewOsFs()

	var file afero.File
	var err error
	if retry {
		file, err = fs.Open(filename)
	} else {
		file, err = fs.Create(filename)
	}
	if err != nil {
		return err
	}

	// Figure out where the previous plotter got interrupted
	var tableIndex, tableStart, tableEnd, headerLen, wrote int
	if retry {
		tableIndex, tableStart, tableEnd, err = getLastTableIndexAndPositions(file)
	} else {
		headerLen, err = WriteHeader(file, k, id)
	}
	if err != nil {
		return err
	}

	start := time.Now()
	if tableIndex == 0 {
		fmt.Println("Computing table 1...")
		wrote, err = WriteFirstTable(file, k, headerLen+1, id)
		if err != nil {
			return err
		}
		fmt.Println("Sorting table 1...")
		if err := sort.OnDisk(file, fs, headerLen+1, wrote+headerLen+1, availableMemory, k, 1); err != nil {
			return err
		}
		if err := updateLastTableIndexAndPositions(file, 1, headerLen+1, wrote+headerLen+1); err != nil {
			return err
		}
		fmt.Printf("F1 calculations finished in %v (wrote %s)\n", time.Since(start), utils.PrettySize(wrote))
	}

	fx, err := NewFx(uint64(k), id)
	if err != nil {
		return err
	}

	var previousStart, currentStart int
	if tableIndex == 0 {
		previousStart = headerLen + 1
		currentStart = headerLen + wrote + 1
		// set table index to 1 so the loop below will
		// work just fine
		tableIndex = 1
	} else {
		fmt.Printf("Restarting plotting process from table %d.\n", tableIndex+1)
		previousStart = tableStart
		currentStart = tableEnd + 1
	}

	for t := tableIndex + 1; t <= 7; t++ {
		start = time.Now()
		fmt.Printf("Computing table %d...\n", t)
		entryLen := serialize.EntrySize(k, t)
		tWrote, err := WriteTable(file, k, t, previousStart, currentStart, entryLen, fx)
		if err != nil {
			return err
		}
		previousStart = currentStart
		currentStart += tWrote + 1

		fmt.Printf("Sorting table %d...\n", t)
		// Remove EOT from entries and currentStart
		if err := sort.OnDisk(file, fs, previousStart, tWrote, availableMemory, k, t); err != nil {
			return err
		}
		if err := updateLastTableIndexAndPositions(file, t, previousStart, previousStart+tWrote); err != nil {
			return err
		}
		fmt.Printf("F%d calculations finished in %v (wrote %s)\n", t, time.Since(start), utils.PrettySize(tWrote))
	}

	return nil
}

func WriteFirstTable(file afero.File, k, start int, id []byte) (int, error) {
	f1, err := NewF1(k, id)
	if err != nil {
		return 0, err
	}

	var wrote int
	maxNumber := uint64(math.Pow(2, float64(k)))

	// TODO: Batch writes
	for x := uint64(0); x < maxNumber; x++ {
		f1x := f1.Calculate(x)
		n, err := serialize.Write(file, int64(start+wrote), f1x, &x, nil, nil, nil, k)
		if err != nil {
			return wrote + n, err
		}
		wrote += n
	}

	eotBytes, err := WriteEOT(file, wrote/int(maxNumber))
	if err != nil {
		return wrote + eotBytes, err
	}
	fmt.Printf("Wrote %d entries (size: %s)\n", maxNumber, utils.PrettySize(wrote))
	return wrote + eotBytes, nil
}

// WriteEOT writes the last entry in the table that should signal
// that we just finished reading the table.
func WriteEOT(file afero.File, entryLen int) (int, error) {
	eotEntry := []byte(serialize.EOT)
	// TODO: newlines are merely added for readability of the plot
	// but readability should not be a goal so remove them eventually
	// and follow the format used in the reference implementation.
	newLine := []byte("\n")
	// entries are supposed to be larger than EOT so we should
	// always prepend bytes here.
	rest := make([]byte, entryLen-len(eotEntry)-len(newLine))
	return file.Write(append(eotEntry, append(rest, newLine...)...))
}

// WriteTable reads the t-1'th table from the file and writes the t'th table.
// The total number of bytes and the amount of entries written is returned.
// Both the total number of bytes and the amount of entries contain EOT as an
// entry so callers can easily estimate the average entry size.
func WriteTable(file afero.File, k, t, previousStart, currentStart, entryLen int, fx *Fx) (int, error) {
	var (
		read    int
		wrote   int
		entries int

		bucketID     uint64
		leftBucketID uint64
		leftBucket   []*serialize.Entry
		rightBucket  []*serialize.Entry
	)

	var index int

	for {
		// Read an entry from the previous table.
		leftEntry, bytesRead, err := serialize.Read(file, int64(previousStart+read), entryLen, k)
		if errors.Is(err, serialize.EOTErr) || errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return wrote, fmt.Errorf("cannot read left entry: %w", err)
		}
		read += bytesRead
		leftEntry.Index = index

		leftBucketID = parameters.BucketID(leftEntry.Fx)
		switch {
		case leftBucketID == bucketID:
			// Add entries in the left bucket
			leftBucket = append(leftBucket, leftEntry)

		case leftBucketID == bucketID+1:
			// Add entries in the right bucket
			rightBucket = append(rightBucket, leftEntry)

		default:
			if len(leftBucket) > 0 && len(rightBucket) > 0 {
				// We have finished adding to both buckets, now we need to compare them.
				// For any matches, we are going to calculate outputs for the next table.
				entries, wrote, err = WriteMatches(file, fx, leftBucket, rightBucket, currentStart, t, k)
				if err != nil {
					return wrote, fmt.Errorf("cannot write matches: %w", err)
				}
			}
			if leftBucketID == bucketID+2 {
				// Keep the right bucket as the new left bucket
				bucketID++
				leftBucket = rightBucket
				rightBucket = nil
			} else {
				// This bucket id is greater than bucketID+2 so we need to
				// start over building both buckets.
				bucketID = leftBucketID
				leftBucket = nil
				rightBucket = nil
			}
		}

		// advance the table index
		index++
	}

	if entries == 0 {
		return wrote, fmt.Errorf("no matches found to write table #%d; try with a larger k", t)
	}

	eotBytes, err := WriteEOT(file, wrote/entries)
	if err != nil {
		return wrote + eotBytes, err
	}
	// we don't really care about including EOT as an entry in the log
	// and the only reason it is returned as part of entries is to allow
	// callers to estimate the average entry size.
	fmt.Printf("Wrote %d entries (size: %s)\n", entries, utils.PrettySize(wrote))

	return wrote + eotBytes, nil
}

var plotHeader = []byte("Proof of Space Plot")

// WriteHeader writes the plot file header to a file
// 19 bytes  - "Proof of Space Plot" (utf-8)
// 32 bytes  - unique plot id
// 1 byte    - k
// 1 byte    - index of the last table that got successfully written, used for re-entrancy
// 8 byte    - start of the last table that got successfully written, used for re-entrancy
// 8 byte    - end of the last table that got successfully written, used for re-entrancy
func WriteHeader(file afero.File, k int, id []byte) (int, error) {
	n, err := file.Write(plotHeader)
	if err != nil {
		return n, err
	}

	nmore, err := file.Write(id)
	n += nmore
	if err != nil {
		return n, err
	}

	nmore, err = file.Write([]byte{byte(k)})
	n += nmore
	if err != nil {
		return n, err
	}

	lastTableIndex := bits.Uint64ToBytes(0, 1)
	nmore, err = file.Write(lastTableIndex)
	n += nmore
	if err != nil {
		return n, err
	}

	lastTableStart := bits.Uint64ToBytes(uint64(nmore+8), 64)
	nmore, err = file.Write(lastTableStart)
	n += nmore
	if err != nil {
		return n, err
	}

	// Ensure we can write indexes even for very large
	// files by using a 64-bit number.
	lastTableEnd := bits.Uint64ToBytes(uint64(nmore+8), 64)
	nmore, err = file.Write(lastTableEnd)
	return n + nmore, err
}

// GetKey returns the key from an existing plot.
func GetKey(plotPath string) ([32]byte, error) {
	const expected = 32
	key := [expected]byte{}

	fs := afero.NewOsFs()
	file, err := fs.Open(plotPath)
	if err != nil {
		return key, fmt.Errorf("cannot open plot file: %w", err)
	}
	read, err := file.ReadAt(key[:], int64(len(plotHeader)))
	if err != nil {
		return key, fmt.Errorf("cannot read plot: %w", err)
	}
	if read != expected {
		return key, fmt.Errorf("expected to read %d bytes, read %d", expected, read)
	}

	return key, nil
}

// getLastTableIndexAndPositions returns the index, start, and end of the
// last table that got successfully plotted.
func getLastTableIndexAndPositions(file afero.File) (int, int, int, error) {
	tableIndexBytes := make([]byte, 1)
	if _, err := file.ReadAt(tableIndexBytes, 53); err != nil {
		return 0, 0, 0, err
	}
	tableStartBytes := make([]byte, 8)
	if _, err := file.ReadAt(tableStartBytes, 54); err != nil {
		return 0, 0, 0, err
	}
	tableEndBytes := make([]byte, 8)
	if _, err := file.ReadAt(tableEndBytes, 62); err != nil {
		return 0, 0, 0, err
	}
	return int(bits.BytesToUint64(tableIndexBytes, 1)),
		int(bits.BytesToUint64(tableStartBytes, 64)),
		int(bits.BytesToUint64(tableEndBytes, 64)),
		nil
}

func updateLastTableIndexAndPositions(file afero.File, index, start, end int) error {
	tableIndexBytes := bits.Uint64ToBytes(uint64(index), 1)
	if _, err := file.WriteAt(tableIndexBytes, 53); err != nil {
		return err
	}
	tableStartBytes := bits.Uint64ToBytes(uint64(start), 64)
	if _, err := file.WriteAt(tableStartBytes, 54); err != nil {
		return err
	}
	tableEndBytes := bits.Uint64ToBytes(uint64(end), 64)
	_, err := file.WriteAt(tableEndBytes, 62)
	return err
}
