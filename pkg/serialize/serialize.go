package serialize

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/spf13/afero"

	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

func Write(file afero.File, offset int64, x, fx uint64, k int) (int, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("cannot set file offset at %d: %v", offset, err)
	}
	// TODO: Batch writes
	// TODO: Write in binary instead of text format (FlatBuffers?)
	src := mybits.Uint64ToBytes(fx, k)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	src = mybits.Uint64ToBytes(x, k)
	xDst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(xDst, src)

	dst = append(dst, ',')
	dst = append(dst, xDst...)
	dst = append(dst, '\n')
	return file.Write(dst)
}

func Read(file afero.File, offset int64, entryLen, k int) (fx uint64, x uint64, err error) {
	e := make([]byte, entryLen)

	if _, err := file.ReadAt(e, offset); err != nil {
		return 0, 0, err
	}

	parts := bytes.Split(e, []byte(","))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid line read: %v", parts)
	}
	// drop delimeter
	parts[1] = bytes.TrimSpace(parts[1])

	dst := make([]byte, hex.DecodedLen(len(parts[0])))
	_, err = hex.Decode(dst, parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("cannot decode f(x): %v", err)
	}
	fx = mybits.BytesToUint64(dst, k)

	dst = make([]byte, hex.DecodedLen(len(parts[1])))
	_, err = hex.Decode(dst, parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("cannot decode x: %v", err)
	}
	x = mybits.BytesToUint64(dst, k)

	return
}
