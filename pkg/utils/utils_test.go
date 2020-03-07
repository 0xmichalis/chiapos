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
		a, b, k int
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
