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

const EOT = "\\0"

var EOTErr = errors.New("EOT")

func writeTo(dst []byte, val uint64, k int) {
	src := mybits.Uint64ToBytes(val, k)
	tmp := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(tmp, src)
	dst = append(dst, tmp...)
}

func Write(file afero.File, offset int64, fx uint64, x, pos, posOffset *uint64, collated *big.Int, k int) (int, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("cannot set file offset at %d: %v", offset, err)
	}
	// TODO: Batch writes
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
		writeTo(dst, *pos, k)
		// posOffset has to be non-nil at this point
		dst = append(dst, ',')
		writeTo(dst, *posOffset, k)
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

func Read(file afero.File, offset int64, entryLen, k int) (*Entry, int, error) {
	e := make([]byte, entryLen)

	read, err := file.ReadAt(e, offset)
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

		// drop delimeter
		parts[1] = bytes.TrimSpace(parts[1])

		dst := make([]byte, hex.DecodedLen(len(parts[0])))
		_, err = hex.Decode(dst, parts[0])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x): %v", err)
		}
		fx := mybits.BytesToUint64(dst, k)

		dst = make([]byte, hex.DecodedLen(len(parts[1])))
		_, err = hex.Decode(dst, parts[1])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode x: %v", err)
		}
		x := mybits.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, X: &x}

	case 3:
		// we are reading the last table

		// drop delimeter
		parts[2] = bytes.TrimSpace(parts[2])

		dst := make([]byte, hex.DecodedLen(len(parts[0])))
		_, err = hex.Decode(dst, parts[0])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x): %v", err)
		}
		fx := mybits.BytesToUint64(dst, k)

		dst = make([]byte, hex.DecodedLen(len(parts[1])))
		_, err = hex.Decode(dst, parts[1])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos: %v", err)
		}
		pos := mybits.BytesToUint64(dst, k)

		dst = make([]byte, hex.DecodedLen(len(parts[2])))
		_, err = hex.Decode(dst, parts[2])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset: %v", err)
		}
		posOffset := mybits.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset}

	case 4:

		// drop delimeter
		parts[3] = bytes.TrimSpace(parts[3])

		dst := make([]byte, hex.DecodedLen(len(parts[0])))
		_, err = hex.Decode(dst, parts[0])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x): %v", err)
		}
		fx := mybits.BytesToUint64(dst, k)

		dst = make([]byte, hex.DecodedLen(len(parts[1])))
		_, err = hex.Decode(dst, parts[1])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos: %v", err)
		}
		pos := mybits.BytesToUint64(dst, k)

		dst = make([]byte, hex.DecodedLen(len(parts[2])))
		_, err = hex.Decode(dst, parts[2])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset: %v", err)
		}
		posOffset := mybits.BytesToUint64(dst, k)

		dst = make([]byte, hex.DecodedLen(len(parts[3])))
		_, err = hex.Decode(dst, parts[3])
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset: %v", err)
		}
		collated := new(big.Int).SetBytes(dst)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset, Collated: collated}

	default:
		return nil, read, fmt.Errorf("invalid line read: %v", parts)
	}

	return entry, read, nil
}
