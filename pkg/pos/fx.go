package pos

import (
	"crypto/cipher"
	"fmt"
	"math/big"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/rraes"
)

type Fx struct {
	k   int
	key cipher.Block
}

func NewFx(k int, key []byte) (*Fx, error) {
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
	at := At(cl, cr, f.k, t, f.key)
	return at ^ fx, nil
}
