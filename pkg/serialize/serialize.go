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

func writeTo(dst []byte, val uint64, k int) []byte {
	src := mybits.Uint64ToBytes(val, k)
	tmp := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(tmp, src)
	return append(dst, tmp...)
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

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %v", fxBytes, err)
		}
		fx := mybits.BytesToUint64(dst, k)

		xBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(xBytes)))
		_, err = hex.Decode(dst, xBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode x (%s): %v", xBytes, err)
		}
		x := mybits.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, X: &x}

	case 3:
		// we are reading the last table

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %v", fxBytes, err)
		}
		fx := mybits.BytesToUint64(dst, k)

		posBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(posBytes)))
		_, err = hex.Decode(dst, posBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos (%s): %v", posBytes, err)
		}
		pos := mybits.BytesToUint64(dst, k)

		posOffsetBytes := preparePart(parts[2])
		dst = make([]byte, hex.DecodedLen(len(posOffsetBytes)))
		_, err = hex.Decode(dst, posOffsetBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset (%s): %v", posOffsetBytes, err)
		}
		posOffset := mybits.BytesToUint64(dst, k)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset}

	case 4:

		fxBytes := preparePart(parts[0])
		dst := make([]byte, hex.DecodedLen(len(fxBytes)))
		_, err = hex.Decode(dst, fxBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode f(x) (%s): %v", fxBytes, err)
		}
		fx := mybits.BytesToUint64(dst, k)

		posBytes := preparePart(parts[1])
		dst = make([]byte, hex.DecodedLen(len(posBytes)))
		_, err = hex.Decode(dst, posBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos (%s): %v", posBytes, err)
		}
		pos := mybits.BytesToUint64(dst, k)

		posOffsetBytes := preparePart(parts[2])
		dst = make([]byte, hex.DecodedLen(len(posOffsetBytes)))
		_, err = hex.Decode(dst, posOffsetBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode pos offset (%s): %v", posOffsetBytes, err)
		}
		posOffset := mybits.BytesToUint64(dst, 10)

		collatedBytes := preparePart(parts[3])
		dst = make([]byte, hex.DecodedLen(len(collatedBytes)))
		_, err = hex.Decode(dst, collatedBytes)
		if err != nil {
			return nil, read, fmt.Errorf("cannot decode collated value (%s): %v", collatedBytes, err)
		}
		collated := new(big.Int).SetBytes(dst)

		entry = &Entry{Fx: fx, Pos: &pos, Offset: &posOffset, Collated: collated}

	default:
		return nil, read, fmt.Errorf("invalid line read: %s", parts)
	}

	return entry, read, nil
}
