package pos

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"time"

	"github.com/spf13/afero"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/serialize"
	"github.com/kargakis/chiapos/pkg/utils"
	"github.com/kargakis/chiapos/pkg/utils/bits"
	"github.com/kargakis/chiapos/pkg/utils/sort"
)

// ForwardPropagate is Phase 1 of the plotter. During this phase, all of the tables,
// and f functions are evaluated. The result is an intermediate plot file, that is
// several times larger than what the final file will be, but that has all of the
// proofs of space in it. First, F1 is computed, which is special since it uses
// AES256, and each encryption provides multiple output values. Then, the rest of the
// f functions are computed, and a sort on disk happens for each table.
func ForwardPropagate(fs afero.Fs, file afero.File, k, availableMemory int, id []byte, retry bool) (int, error) {
	// Figure out where the previous plotter got interrupted
	var tableIndex, tableStart, tableEnd, headerLen, wrote int
	var err error

	if retry {
		tableIndex, tableStart, tableEnd, err = getLastTableIndexAndPositions(file)
	} else {
		fmt.Printf("Generating plot at %s with k=%d\n", file.Name(), k)
		headerLen, err = WriteHeader(file, k, id)
	}
	if err != nil {
		return headerLen, err
	}

	start := time.Now()
	if tableIndex == 0 {
		fmt.Println("Computing table 1...")
		wrote, err = WriteFirstTable(file, k, headerLen+1, id)
		if err != nil {
			return wrote, err
		}
		fmt.Println("Sorting table 1...")
		if err := sort.OnDisk(file, fs, headerLen+1, wrote+headerLen+1, availableMemory, k, 1); err != nil {
			return wrote, err
		}
		if err := updateLastTableIndexAndPositions(file, 1, headerLen+1, wrote+headerLen+1); err != nil {
			return wrote, err
		}
		fmt.Printf("F1 calculations finished in %v (wrote %s)\n", time.Since(start), utils.PrettySize(wrote))
	}

	fx, err := NewFx(k, id)
	if err != nil {
		return wrote, err
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
		tWrote, err := WriteTable(file, k, t, previousStart, currentStart, fx)
		if err != nil {
			return tWrote + wrote, err
		}
		wrote += tWrote
		previousStart = currentStart
		currentStart += tWrote + 1

		fmt.Printf("Sorting table %d...\n", t)
		// Remove EOT from entries and currentStart
		if err := sort.OnDisk(file, fs, previousStart, tWrote, availableMemory, k, t); err != nil {
			return wrote, err
		}
		if err := updateLastTableIndexAndPositions(file, t, previousStart, previousStart+tWrote); err != nil {
			return wrote, err
		}
		fmt.Printf("F%d calculations finished in %v (wrote %s)\n", t, time.Since(start), utils.PrettySize(tWrote))
	}

	return wrote, nil
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
		f1x := f1.CalculateOne(x)
		n, err := serialize.Write(file, int64(start+wrote), f1x, &x, nil, nil, nil, k)
		if err != nil {
			return wrote + n, err
		}
		wrote += n
	}

	eotBytes, err := WriteEOT(file, wrote/int(maxNumber))
	return wrote + eotBytes, err
}

// WriteEOT writes the last entry in the table that should signal
// that we just finished reading the table.
func WriteEOT(file afero.File, entryLen int) (int, error) {
	eotEntry := []byte(serialize.EOT)
	delimiter := []byte{serialize.EntriesDelimiter}
	// prepend the same amount of bytes an entry has to the
	// delimiter. TODO: Stop doing this?
	rest := make([]byte, entryLen-len(eotEntry)-len(delimiter))
	return file.Write(append(eotEntry, append(rest, delimiter...)...))
}

// WriteTable reads the t-1'th table from the file and writes the t'th table.
// The total number of bytes and the amount of entries written is returned.
// Both the total number of bytes and the amount of entries contain EOT as an
// entry so callers can easily estimate the average entry size.
func WriteTable(file afero.File, k, t, previousStart, currentStart int, fx *Fx) (int, error) {
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
	entryLen := serialize.EntrySize(k, t)

	for {
		// Read an entry from the previous table.
		leftEntry, bytesRead, err := serialize.Read(file, int64(previousStart+read), entryLen, k)
		if errors.Is(err, serialize.EOTErr) || errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return wrote, fmt.Errorf("cannot read left entry: %w", err)
		}
		leftEntry.Index = previousStart + read
		read += bytesRead

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
				matches := FindMatches(leftBucket, rightBucket)
				for _, m := range matches {
					le, re := m.Left, m.Right
					var leftMetadata, rightMetadata *big.Int
					if le.X != nil {
						leftMetadata = big.NewInt(int64(*le.X))
					} else if le.Collated != nil {
						leftMetadata = le.Collated
					}
					if re.X != nil {
						rightMetadata = big.NewInt(int64(*re.X))
					} else if re.Collated != nil {
						rightMetadata = re.Collated
					}

					f, err := fx.Calculate(t, le.Fx, leftMetadata, rightMetadata)
					if err != nil {
						return wrote, err
					}
					// This is the collated output stored next to the entry - useful
					// for generating outputs for the next table.
					collated, err := Collate(t, k, leftMetadata, rightMetadata)
					if err != nil {
						return wrote, err
					}
					// Now write the new output in the next table.
					index := uint64(le.Index)
					offset := uint64(re.Index - le.Index)
					w, err := serialize.Write(file, int64(currentStart+wrote), f, nil, &index, &offset, collated, k)
					if err != nil {
						return wrote, err
					}
					entries++
					wrote += w
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
	return wrote + eotBytes, err
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

	kBytes := bits.Uint64ToBytes(uint64(k), 1)
	nmore, err = file.Write(kBytes)
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
func GetKey(plotPath string) ([]byte, error) {
	key := make([]byte, utils.KeyLen)

	fs := afero.NewOsFs()
	file, err := fs.Open(plotPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open plot file: %w", err)
	}
	read, err := file.ReadAt(key, int64(len(plotHeader)))
	if err != nil {
		return nil, fmt.Errorf("cannot read plot: %w", err)
	}
	if read != utils.KeyLen {
		return nil, fmt.Errorf("expected to read %d bytes, read %d", utils.KeyLen, read)
	}

	return key, nil
}

// getLastTableIndexAndPositions returns the index, start, and end of the
// last table that got successfully plotted.
func getLastTableIndexAndPositions(file afero.File) (int, int, int, error) {
	start := int64(len(plotHeader) + utils.KeyLen + 1)

	tableIndexBytes := make([]byte, 1)
	read, err := file.ReadAt(tableIndexBytes, start)
	if err != nil {
		return 0, 0, 0, err
	}

	tableStartBytes := make([]byte, 8)
	more, err := file.ReadAt(tableStartBytes, start+int64(read))
	if err != nil {
		return 0, 0, 0, err
	}
	read += more

	tableEndBytes := make([]byte, 8)
	if _, err := file.ReadAt(tableEndBytes, start+int64(read)); err != nil {
		return 0, 0, 0, err
	}

	return int(bits.BytesToUint64(tableIndexBytes, 1)),
		int(bits.BytesToUint64(tableStartBytes, 64)),
		int(bits.BytesToUint64(tableEndBytes, 64)),
		nil
}

func updateLastTableIndexAndPositions(file afero.File, index, tableStart, tableEnd int) error {
	start := int64(len(plotHeader) + utils.KeyLen + 1)

	tableIndexBytes := bits.Uint64ToBytes(uint64(index), 1)
	wrote, err := file.WriteAt(tableIndexBytes, start)
	if err != nil {
		return err
	}

	tableStartBytes := bits.Uint64ToBytes(uint64(tableStart), 64)
	more, err := file.WriteAt(tableStartBytes, start+int64(wrote))
	if err != nil {
		return err
	}
	wrote += more

	tableEndBytes := bits.Uint64ToBytes(uint64(tableEnd), 64)
	_, err = file.WriteAt(tableEndBytes, start+int64(wrote))
	return err
}
