package serialize

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/spf13/afero"

	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

const EOT = "\\0"

var EOTErr = errors.New("EOT")

type Entry struct {
	Fx uint64
	X  *uint64

	// Position of the left match in the previous table
	// TODO: This should be k+1 bits.
	Pos *uint64
	// Offset to find the right match in the previous table
	// TODO: This should be a 10-bit offset.
	Offset *uint64
	// Collated value to be used as input in the next table.
	Collated *big.Int

	// Index of the f output inside the table
	Index int
}

type ByOutput struct {
	Entries    []*Entry
	TableIndex int
}

func (b ByOutput) Len() int      { return len(b.Entries) }
func (b ByOutput) Swap(i, j int) { b.Entries[i], b.Entries[j] = b.Entries[j], b.Entries[i] }
func (b ByOutput) Less(i, j int) bool {
	e := b.Entries

	// Sort first and last table based on their outputs only.
	if e[i].Fx != e[j].Fx || b.TableIndex == 1 || b.TableIndex == 7 {
		return e[i].Fx < e[j].Fx
	}

	// If we are sorting any other than the first and last tables
	// then we should also take into account positions and offsets.
	if *e[i].Pos != *e[j].Pos {
		return *e[i].Pos < *e[j].Pos
	}
	return *e[i].Offset < *e[j].Offset
}

func writeTo(dst []byte, val uint64, k int) []byte {
	src := mybits.Uint64ToBytes(val, k)
	tmp := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(tmp, src)
	return append(dst, tmp...)
}

func Write(file afero.File, offset int64, fx uint64, x, pos, posOffset *uint64, collated *big.Int, k int) (int, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("cannot set file offset at %d: %w", offset, err)
	}
	// TODO: Write in binary instead of text format (FlatBuffers?)
	src := mybits.Uint64ToBytes(fx, k)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	if x != nil {
		src = mybits.Uint64ToBytes(*x, k)
		xDst := make([]byte, hex.EncodedLen(len(src)))
		hex.Encode(xDst, src)
		dst = append(dst, ',')
		dst = append(dst, xDst...)
	}

	// Write the pos,offset if we are provided one
	if pos != nil {
		dst = append(dst, ',')
		dst = writeTo(dst, *pos, k)
		// posOffset has to be non-nil at this point
		dst = append(dst, ',')
		dst = writeTo(dst, *posOffset, 10)
	}
	// Write the collated value if we are provided one
	if collated != nil {
		serialized := collated.Bytes()
		sDst := make([]byte, hex.EncodedLen(len(serialized)))
		hex.Encode(sDst, serialized)
		dst = append(dst, ',')
		dst = append(dst, sDst...)
	}

	dst = append(dst, '\n')
	return file.Write(dst)
}

func preparePart(part []byte) []byte {
	// TODO: This is ugly and should be fixed in a different way
	return bytes.TrimSpace(bytes.TrimRight(part, ","))
}

// read ensures all bytes up to the delimeter will be read.
// If more bytes are read, the extra bytes are dropped.
// If less bytes are read, one more read is performed which
// should include the next delimeter.
func read(file afero.File, offset int64, delimeter []byte, entryLen int) (int, []byte, error) {
	e := make([]byte, entryLen)

	read, err := file.ReadAt(e, offset)
	if err != nil {
		return read, nil, err
	}

	delimeterIndex := bytes.Index(e, delimeter)
	if delimeterIndex == -1 {
		// If there is no delimeter we need to read more.
		// One more read of entryLen bytes should suffice.
		additional := make([]byte, entryLen)
		more, err := file.ReadAt(additional, offset+int64(read))
		if err != nil {
			return read + more, nil, err
		}
		delimeterIndex = bytes.Index(additional, delimeter)
		e = append(e, additional[:delimeterIndex+1]...)
		return len(e), e, nil
	}

	// if we got a delimeter in our read bytes, it is either in the end
	// of the byte slice (normal case), somewhere in between (collated
	// value is size is not fixed for some reason), or at the start (bad read).
	read, e = dropDelimeters(file, e, delimeter)
	return read, e, nil
}

func dropDelimeters(file afero.File, e, delimeter []byte) (int, []byte) {
	delimeterIndex := bytes.Index(e, delimeter)
	switch delimeterIndex {

	case 0:
		e = bytes.TrimLeft(e, string(delimeter))
		// There may be more than one delimeter as part of this entry...
		var read int
		read, e = dropDelimeters(file, e, delimeter)
		return read + 1, e

	case len(e):
		// normal case; do nothing

	default:
		e = e[:delimeterIndex+1]
	}
	return len(e), e
}

func Read(file afero.File, offset int64, entryLen, k int) (*Entry, int, error) {
	// HACK: collated values unfortunately can break the assumption
	// that all entries have fixed length so if our entry contains
	// a delimeter not at the end of the entry, then we need to drop
	// what we read up to the newline.
	read, e, err := read(file, offset, []byte("\n"), entryLen)
	if err != nil {
		return nil, read, err
	}

	if bytes.Contains(e, []byte(EOT)) {
		return nil, read, EOTErr
	}

	var entry *Entry
	parts := bytes.Split(e, []byte(","))

	switch len(parts) {
	case 2:
		// we are reading the first table

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
		}
		fx := mybits.BytesToUint64(dst, k)

		xBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(xBytes)))
		_, err = hex.Decode(dst, xBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode x (%s): %w", xBytes, err)
		}
		x := mybits.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, X: &x}

	case 3:
		// we are reading the last table

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
		}
		fx := mybits.BytesToUint64(dst, k)

		posBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(posBytes)))
		_, err = hex.Decode(dst, posBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos (%s): %w", posBytes, err)
		}
		pos := mybits.BytesToUint64(dst, k)

		posOffsetBytes := preparePart(parts[2])
		dst = make([]byte, hex.DecodedLen(len(posOffsetBytes)))
		_, err = hex.Decode(dst, posOffsetBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset (%s): %w", posOffsetBytes, err)
		}
		posOffset := mybits.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset}

	case 4:

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
		}
		fx := mybits.BytesToUint64(dst, k)

		posBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(posBytes)))
		_, err = hex.Decode(dst, posBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos (%s): %w", posBytes, err)
		}
		pos := mybits.BytesToUint64(dst, k)

		posOffsetBytes := preparePart(parts[2])
		dst = make([]byte, hex.DecodedLen(len(posOffsetBytes)))
		_, err = hex.Decode(dst, posOffsetBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset (%s): %w", posOffsetBytes, err)
		}
		posOffset := mybits.BytesToUint64(dst, 10)

		collatedBytes := preparePart(parts[3])
		dst = make([]byte, hex.DecodedLen(len(collatedBytes)))
		_, err = hex.Decode(dst, collatedBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode collated value (%s): %w", collatedBytes, err)
		}
		collated := new(big.Int).SetBytes(dst)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset, Collated: collated}

	default:
		return nil, read, fmt.Errorf("invalid line read: %s", parts)
	}

	return entry, read, nil
}
