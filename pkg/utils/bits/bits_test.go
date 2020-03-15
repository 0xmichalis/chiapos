package bits_test

import (
	"math/big"
	"math/bits"
	"testing"

	bitsutil "github.com/kargakis/chiapos/pkg/utils/bits"
)

func TestUint64ToBytes(t *testing.T) {
	var n uint64 = 1000
	nBytes := bitsutil.Uint64ToBytes(n, 15)
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

	n := bitsutil.BytesToUint64(nBytes, 64)
	if bigN.Uint64() != n {
		t.Errorf("expected n to be %d, got %d", bigN.Uint64(), n)
	}
}

func TestIsAtMostKBits(t *testing.T) {
	tests := []struct {
		name   string
		x      uint64
		k      uint64
		expect bool
	}{
		{
			name:   "is",
			x:      uint64(bits.Reverse32(1)),
			k:      32,
			expect: true,
		},
		{
			name:   "is not",
			x:      uint64(bits.Reverse64(1)),
			k:      32,
			expect: false,
		},
	}

	for _, tt := range tests {
		got := bitsutil.IsAtMostKBits(tt.x, tt.k)
		if got != tt.expect {
			t.Errorf("%s: expected %t, got %t (bit length: %d)", tt.name, tt.expect, got, bits.Len64(tt.x))
		}
	}
}

func TestNormalise(t *testing.T) {
	tests := []struct {
		x      uint64
		k      uint64
		expect uint64
	}{
		{
			x:      uint64(bits.Reverse32(1)),
			k:      32,
			expect: uint64(bits.Reverse32(1)),
		},
		{
			x:      bits.Reverse64(1),
			k:      32,
			expect: uint64(bits.Reverse32(1)),
		},
	}

	for i, tt := range tests {
		got := bitsutil.Normalise(tt.x, tt.k)
		if got != tt.expect {
			t.Errorf("tc #%d: expected %d (bit length: %d), got %d (bit length: %d)", i+1, tt.expect, bits.Len64(tt.expect), got, bits.Len64(got))
		}
	}
}
