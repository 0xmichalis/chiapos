package serialize

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/spf13/afero"

	"github.com/kargakis/chiapos/pkg/parameters"
	bitsutil "github.com/kargakis/chiapos/pkg/utils/bits"
)

const (
	// End of Table special character
	EOT = "\\0"

	// size of the position offset in bits
	posOffsetSize = 32

	posBitSize = 64

	// entriesDelimiter is a delimiter used to separate entries
	EntriesDelimiter = '\n'

	// entryDelimiter is a delimiter used to separate different
	// parts of a single entry
	entryDelimiter = ','
)

var EOTErr = errors.New("EOT")

type Entry struct {
	Fx uint64
	X  *uint64

	// Position of the left match in the previous table
	// This should be k+1 bits.
	Pos *uint64
	// Offset to find the right match in the previous table
	// This should be a 10-bit offset.
	Offset *uint64
	// Collated value to be used as input in the next table.
	Collated *big.Int

	// Index of the f output inside the plot.
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

// CollaSize returns the collation size for t.
func CollaSize(t int) int {
	var size int
	switch t {
	case 2:
		size = 1
	case 3, 7:
		size = 2
	case 4, 5:
		size = 4
	case 6:
		size = 3
	}
	return size
}

func writeTo(dst []byte, val uint64, k int) []byte {
	src := bitsutil.Uint64ToBytes(val, k)
	tmp := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(tmp, src)
	return append(dst, tmp...)
}

// Write serializes a table entry in file.
func Write(file afero.File, offset int64, fx uint64, x, pos, posOffset *uint64, collated *big.Int, k int) (int, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("cannot set file offset at %d: %w", offset, err)
	}
	// TODO: Write in binary instead of text format (FlatBuffers?)
	src := bitsutil.Uint64ToBytes(fx, k+parameters.ParamEXT)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	if x != nil {
		src = bitsutil.Uint64ToBytes(*x, k)
		xDst := make([]byte, hex.EncodedLen(len(src)))
		hex.Encode(xDst, src)
		dst = append(dst, entryDelimiter)
		dst = append(dst, xDst...)
	}

	if pos != nil {
		dst = append(dst, entryDelimiter)
		// Store positions to previous tables, in k+1 bits. This is because we may have
		// more than 2^k entries in some of the tables, so we need an extra bit.
		dst = writeTo(dst, *pos, posBitSize)
	}
	if posOffset != nil {
		dst = append(dst, entryDelimiter)
		dst = writeTo(dst, *posOffset, posOffsetSize)
	}

	// Write the collated value if we are provided one
	if collated != nil {
		serialized := collated.Bytes()
		sDst := make([]byte, hex.EncodedLen(len(serialized)))
		hex.Encode(sDst, serialized)
		dst = append(dst, entryDelimiter)
		dst = append(dst, sDst...)
	}

	dst = append(dst, EntriesDelimiter)
	return file.Write(dst)
}

func preparePart(part []byte) []byte {
	return bytes.TrimRight(bytes.TrimRight(part, string(EntriesDelimiter)), string(entryDelimiter))
}

// read ensures all bytes up to the delimiter will be read.
// If more bytes are read, the extra bytes are dropped.
// If less bytes are read, one more read is performed which
// should include the next delimiter.
func read(file afero.File, offset int64, delimiter []byte, entryLen int) (int, []byte, error) {
	e := make([]byte, entryLen)

	read, err := file.ReadAt(e, offset)
	if err != nil {
		return read, nil, err
	}

	delimiterIndex := bytes.Index(e, delimiter)
	if delimiterIndex == -1 {
		// If there is no delimiter we need to read more.
		// One more read of entryLen bytes should suffice.
		additional := make([]byte, entryLen)
		more, err := file.ReadAt(additional, offset+int64(read))
		if err != nil {
			return read + more, nil, err
		}
		delimiterIndex = bytes.Index(additional, delimiter)
		e = append(e, additional[:delimiterIndex+1]...)
		return len(e), e, nil
	}

	// if we got a delimiter in our read bytes, it is either in the end
	// of the byte slice (normal case), somewhere in between (collated
	// value size is not fixed for some reason), or at the start (bad read).
	read, e = dropDelimiters(file, e, delimiter)
	return read, e, nil
}

func dropDelimiters(file afero.File, e, delimiter []byte) (int, []byte) {
	delimiterIndex := bytes.Index(e, delimiter)
	switch delimiterIndex {

	case 0:
		e = bytes.TrimLeft(e, string(delimiter))
		// There may be more than one delimiter as part of this entry...
		var read int
		read, e = dropDelimiters(file, e, delimiter)
		return read + 1, e

	case len(e):
		// normal case; do nothing

	default:
		e = e[:delimiterIndex+1]
	}
	return len(e), e
}

func Read(file afero.File, offset int64, entryLen, k int) (*Entry, int, error) {
	// HACK: collated values unfortunately can break the assumption
	// that all entries have fixed length so if our entry contains
	// a delimiter not at the end of the entry, then we need to drop
	// what we read up to the newline.
	read, e, err := read(file, offset, []byte{EntriesDelimiter}, entryLen)
	if err != nil {
		return nil, read, err
	}

	if bytes.Contains(e, []byte(EOT)) {
		return nil, read, EOTErr
	}

	var entry *Entry
	parts := bytes.Split(e, []byte{entryDelimiter})

	switch len(parts) {
	case 2:
		// we are reading the first table

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
		}
		fx := bitsutil.BytesToUint64(dst, k+parameters.ParamEXT)

		xBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(xBytes)))
		_, err = hex.Decode(dst, xBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode x (%s): %w", xBytes, err)
		}
		x := bitsutil.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, X: &x}

	case 3:
		// we are reading the last table

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
		}
		fx := bitsutil.BytesToUint64(dst, k+parameters.ParamEXT)

		posBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(posBytes)))
		_, err = hex.Decode(dst, posBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos (%s): %w", posBytes, err)
		}
		pos := bitsutil.BytesToUint64(dst, posBitSize)

		posOffsetBytes := preparePart(parts[2])
		dst = make([]byte, hex.DecodedLen(len(posOffsetBytes)))
		_, err = hex.Decode(dst, posOffsetBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset (%s): %w", posOffsetBytes, err)
		}
		posOffset := bitsutil.BytesToUint64(dst, posOffsetSize)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset}

	case 4:

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
		}
		fx := bitsutil.BytesToUint64(dst, k+parameters.ParamEXT)

		posBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(posBytes)))
		_, err = hex.Decode(dst, posBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos (%s): %w", posBytes, err)
		}
		pos := bitsutil.BytesToUint64(dst, posBitSize)

		posOffsetBytes := preparePart(parts[2])
		dst = make([]byte, hex.DecodedLen(len(posOffsetBytes)))
		_, err = hex.Decode(dst, posOffsetBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset (%s): %w", posOffsetBytes, err)
		}
		posOffset := bitsutil.BytesToUint64(dst, posOffsetSize)

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

func ReadCheckpoint(buf *bufio.Reader, k int) (*Entry, error) {
	read, err := buf.ReadBytes(EntriesDelimiter)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(read, []byte(EOT)) {
		return nil, EOTErr
	}
	parts := bytes.Split(read, []byte{entryDelimiter})
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid entry: %s", string(read))
	}

	fxBytes := preparePart(parts[0])
	dst := make([]byte, hex.DecodedLen(len(fxBytes)))
	_, err = hex.Decode(dst, fxBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot decode f(x) (%s): %w", fxBytes, err)
	}
	fx := bitsutil.BytesToUint64(dst, k+parameters.ParamEXT)

	posBytes := preparePart(parts[1])
	dst = make([]byte, hex.DecodedLen(len(posBytes)))
	_, err = hex.Decode(dst, posBytes)
	if err != nil {
		return nil, fmt.Errorf("cannot decode pos (%s): %w", posBytes, err)
	}
	pos := bitsutil.BytesToUint64(dst, posBitSize)

	return &Entry{Fx: fx, Pos: &pos}, nil
}

// EntrySize returns the expected entry size depending
// on the space parameter k and the table index t.
func EntrySize(k, t int) int {
	xBytes := bitsutil.ToBytes(k)
	fxBytes := bitsutil.ToBytes(k + parameters.ParamEXT)
	posBytes := bitsutil.ToBytes(posBitSize)
	offsetBytes := bitsutil.ToBytes(posOffsetSize)
	collBytes := bitsutil.ToBytes(CollaSize(t) * k)

	switch t {
	case 1:
		// fx + entryDelimiter + x + entriesDelimiter
		return 2*fxBytes + 1 + 2*xBytes + 1
	case 2, 3, 4, 5, 6:
		// fx + entryDelimiter + pos + entryDelimiter + posOffset + entryDelimiter + collated + entriesDelimiter
		return 2*fxBytes + 1 + 2*posBytes + 1 + 2*offsetBytes + 1 + 2*collBytes + 1
	case 7:
		// fx + entryDelimiter + pos + entryDelimiter + posOffset + entriesDelimiter
		return 2*fxBytes + 1 + 2*posBytes + 1 + 2*offsetBytes + 1
	}
	return 0
}
