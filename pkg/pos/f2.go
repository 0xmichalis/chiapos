package pos

import (
	"crypto/cipher"
	"fmt"

	"github.com/kargakis/gochia/pkg/rraes"
)

type F2 struct {
	k   uint64
	key cipher.Block
}

func NewF2(k uint64, key []byte) (*F2, error) {
	if k < kMinPlotSize || k > kMaxPlotSize {
		return nil, fmt.Errorf("invalid k: %d, valid range: %d - %d", k, kMinPlotSize, kMaxPlotSize)
	}

	f2 := &F2{
		k: k,
	}

	aesKey := make([]byte, 16)
	copy(aesKey, key)

	block, err := rraes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	f2.key = block

	return f2, nil
}

func (f *F2) Calculate(x1, x2 uint64) uint64 {
	return 0
}
