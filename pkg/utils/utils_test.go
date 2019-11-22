package utils

import (
	"math/big"
	"math/bits"
	"testing"
)

func TestConcat(t *testing.T) {
	zeroInt := big.NewInt(0)
	tests := []struct {
		name string
		x, y uint64
		k    uint64
		want *big.Int
	}{
		{
			name: "concat",
			x:    4,
			y:    4,
			k:    3,
			want: big.NewInt(36),
		},
		{
			name: "concat will not truncate to k",
			x:    bits.Reverse64(1),
			y:    bits.Reverse64(1),
			k:    57,
			want: zeroInt.SetBit(zeroInt, 120, 1).SetBit(zeroInt, 63, 1),
		},
	}

	for _, tt := range tests {
		got := Concat(tt.k, tt.x, tt.y)
		if got.Cmp(tt.want) != 0 {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestTrunc(t *testing.T) {
	tests := []struct {
		name    string
		x       *big.Int
		a, b, k uint64
		want    *big.Int
	}{
		{
			name: "simple truncate",

			x: big.NewInt(36),
			b: 2,
			k: 6,

			want: big.NewInt(2),
		},
		{
			name: "slice both ends",

			x: big.NewInt(36),
			a: 2,
			b: 5,
			k: 6,

			want: big.NewInt(2),
		},
	}

	for _, tt := range tests {
		got := Trunc(tt.x, tt.a, tt.b, tt.k)
		if got.Cmp(tt.want) != 0 {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
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
		got := IsAtMostKBits(tt.x, tt.k)
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
			x:      uint64(bits.Reverse64(1)),
			k:      32,
			expect: uint64(bits.Reverse32(1)),
		},
	}

	for i, tt := range tests {
		got := Normalise(tt.x, tt.k)
		if got != tt.expect {
			t.Errorf("tc #%d: expected %d (bit length: %d), got %d (bit length: %d)", i+1, tt.expect, bits.Len64(tt.expect), got, bits.Len64(got))
		}
	}
}
