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

func TestBytesToUint64(t *testing.T) {
	bigN := big.NewInt(10000000000)
	nBytes := make([]byte, 8)
	bigNBytes := bigN.Bytes()
	// TODO: Zero-padding needs to move into an utility
	for i := 8 - len(bigNBytes); i < 8; i++ {
		nBytes[i] = bigNBytes[i-(8-len(bigNBytes))]
	}

	n := bits.BytesToUint64(nBytes)
	if bigN.Uint64() != n {
		t.Errorf("expected n to be %d, got %d", bigN.Uint64(), n)
	}
}
