package pos

import (
	"crypto/cipher"
	"fmt"
	"math"
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

func (f *Fx) Calculate(t int, fx uint64, xs ...uint64) uint64 {
	xLen := int(math.Pow(2, float64(t-1)))
	if len(xs) != xLen {
		panic(fmt.Sprintf("expected %d xs, got %d", xLen, len(xs)))
	}

	var cl, cr *big.Int

	switch t {
	case 2:
		cl, cr = big.NewInt(int64(xs[0])), big.NewInt(int64(xs[1]))

	default:
		var err error
		mid := xLen / 2

		cl, err = Ct(t, f.k, xs[0:mid]...)
		if err != nil {
			panic(err)
		}
		cr, err = Ct(t, f.k, xs[mid:len(xs)-1]...)
		if err != nil {
			panic(err)
		}
	}

	at, err := At(cl, cr, f.k, t, f.key)
	if err != nil {
		panic(err)
	}

	return at ^ fx
}
