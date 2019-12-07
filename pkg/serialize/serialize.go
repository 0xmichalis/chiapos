package serialize

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/spf13/afero"

	mybits "github.com/kargakis/gochia/pkg/utils/bits"
)

func Write(file afero.File, offset int64, x, fx uint64) (int, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, fmt.Errorf("cannot set file offset at %d: %v", offset, err)
	}
	// TODO: Batch writes
	// TODO: Write in binary instead of text format (FlatBuffers?)
	src := mybits.Uint64ToBytes(fx)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)

	src = mybits.Uint64ToBytes(x)
	xDst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(xDst, src)

	dst = append(dst, ',')
	dst = append(dst, xDst...)
	dst = append(dst, '\n')
	return file.Write(dst)
}
