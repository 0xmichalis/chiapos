package pos

import (
	"fmt"
	"io"
	"math"
	"time"

	"github.com/spf13/afero"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/serialize"
	"github.com/kargakis/gochia/pkg/utils"
	"github.com/kargakis/gochia/pkg/utils/sort"
)

var AppFs = afero.NewOsFs()

// This is Phase 1, or forward propagation. During this phase, all of the 7 tables,
// and f functions, are evaluated. The result is an intermediate plot file, that is
// several times larger than what the final file will be, but that has all of the
// proofs of space in it. First, F1 is computed, which is special since it uses
// AES256, and each encryption provides multiple output values. Then, the rest of the
// f functions are computed, and a sort on disk happens for each table.
func WritePlotFile(filename string, k, availableMemory int, memo, id []byte) error {
	file, err := AppFs.Create(filename)
	if err != nil {
		return err
	}

	headerLen, err := WriteHeader(file, k, memo, id)
	if err != nil {
		return err
	}

	fmt.Println("Computing table 1...")
	start := time.Now()
	wrote, err := WriteFirstTable(file, k, headerLen, id)
	if err != nil {
		return err
	}

	// if we know beforehand there is not enough space
	// to sort in memory, we can prepare the spare file
	var spare afero.File
	if wrote > availableMemory {
		spare, err = AppFs.Create(filename + "-spare")
		if err != nil {
			return err
		}
	}

	fmt.Println("Sorting table 1...")
	maxNumber := int(math.Pow(2, float64(k)))
	entryLen := wrote / maxNumber
	if err := sort.OnDisk(file, spare, headerLen, wrote+headerLen, availableMemory, entryLen, maxNumber, k); err != nil {
		return err
	}
	fmt.Printf("F1 calculations finished in %v (wrote %s)\n", time.Since(start), utils.PrettySize(wrote))

	fx, err := NewFx(uint64(k), id)
	if err != nil {
		return err
	}

	previousStart := headerLen
	currentStart := headerLen + wrote
	for t := 2; t <= 7; t++ {
		start = time.Now()
		fmt.Printf("Computing table %d...\n", t)
		tWrote, entries, err := WriteTable(file, k, t, previousStart, currentStart, entryLen, fx)
		if err != nil {
			return err
		}
		previousStart = currentStart
		currentStart += tWrote
		entryLen = tWrote / entries

		fmt.Printf("Sorting table %d...\n", t)
		// Remove EOT from entries and currentStart
		if err := sort.OnDisk(file, spare, previousStart, currentStart-entryLen, availableMemory, entryLen+1, entries-1, k); err != nil {
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
func WriteTable(file afero.File, k, t, previousStart, currentStart, entryLen int, fx *Fx) (int, int, error) {
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
		// Read an entry
		leftEntry, bytesRead, err := serialize.Read(file, int64(previousStart+read), entryLen+1, k)
		if err == serialize.EOTErr || err == io.EOF {
			break
		}
		if err != nil {
			return wrote, entries, fmt.Errorf("cannot read left entry: %v", err)
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
				for _, m := range FindMatches(leftBucket, rightBucket) {
					f, err := fx.Calculate(t, m.Left, m.LeftMetadata, m.RightMetadata)
					if err != nil {
						return wrote, entries, err
					}
					// This is the collated output stored next to the entry - useful
					// for generating outputs for the next table.
					collated, err := Collate(t, uint64(k), m.LeftMetadata, m.RightMetadata)
					if err != nil {
						return wrote, entries, err
					}
					// Now write the new output in the next table.
					w, err := serialize.Write(file, int64(currentStart+wrote), f, nil, &m.LeftPosition, &m.Offset, collated, k)
					if err != nil {
						return wrote + w, entries, err
					}
					wrote += w
					entries++
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
		return wrote, 0, fmt.Errorf("no matches found to write table #%d; try with a larger k", t)
	}

	eotBytes, err := WriteEOT(file, wrote/entries)
	if err != nil {
		return wrote + eotBytes, entries, err
	}
	// we don't really care about including EOT as an entry in the log
	// and the only reason it is returned as part of entries is to allow
	// callers to estimate the average entry size.
	fmt.Printf("Wrote %d entries (size: %s)\n", entries, utils.PrettySize(wrote))

	return wrote + eotBytes, entries + 1, nil
}

// WriteHeader writes the plot file header to a file
// 19 bytes  - "Proof of Space Plot" (utf-8)
// 32 bytes  - unique plot id
// 1 byte    - k
// 2 bytes   - memo length
// x bytes   - memo
func WriteHeader(file afero.File, k int, memo, id []byte) (int, error) {
	n, err := file.Write([]byte("Proof of Space Plot"))
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

	sizeBuf := make([]byte, 2)
	sizeBuf[0] = byte(len(memo))
	nmore, err = file.Write(sizeBuf)
	n += nmore
	if err != nil {
		return n, err
	}

	nmore, err = file.Write(memo)
	return n + nmore, err
}
