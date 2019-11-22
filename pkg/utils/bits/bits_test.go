package bits_test

import (
	"math/big"
	"testing"

	"github.com/kargakis/gochia/pkg/utils/bits"
)

func TestUint64ToBytes(t *testing.T) {
	var n uint64 = 1000
	nBytes := bits.Uint64ToBytes(n)
	bigN := new(big.Int).SetBytes(nBytes)
	if bigN.Uint64() != n {
		t.Errorf("expected big.Int(n) to be %d, got %d", n, bigN.Uint64())
	}

}
