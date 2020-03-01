package pos

import (
	"crypto/cipher"
	"fmt"
	"math/big"

	"github.com/kargakis/gochia/pkg/parameters"
	"github.com/kargakis/gochia/pkg/rraes"
)

type Fx struct {
	k   uint64
	key cipher.Block
}

func NewFx(k uint64, key []byte) (*Fx, error) {
	if k < parameters.KMinPlotSize || k > parameters.KMaxPlotSize {
		return nil, fmt.Errorf("invalid k: %d, valid range: %d - %d", k, parameters.KMinPlotSize, parameters.KMaxPlotSize)
	}

	fx := &Fx{
		k: k,
	}

	aesKey := make([]byte, 16)
	copy(aesKey, key)

	block, err := rraes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	fx.key = block

	return fx, nil
}

func (f *Fx) Calculate(t int, fx uint64, cl, cr *big.Int) (uint64, error) {
	at, err := At(cl, cr, f.k, t, f.key)
	if err != nil {
		return 0, fmt.Errorf("cannot generate output via AES encryption: %w", err)
	}

	return at ^ fx, nil
}
