package pos

import (
	"crypto/cipher"
	"fmt"

	"github.com/kargakis/gochia/pkg/rraes"
)

type Fx struct {
	k   uint64
	key cipher.Block
}

func NewFx(k uint64, key []byte) (*Fx, error) {
	if k < kMinPlotSize || k > kMaxPlotSize {
		return nil, fmt.Errorf("invalid k: %d, valid range: %d - %d", k, kMinPlotSize, kMaxPlotSize)
	}

	f1 := &Fx{
		k: k,
	}

	aesKey := make([]byte, 16)
	copy(aesKey, key)

	block, err := rraes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	f1.key = block

	return f1, nil
}
