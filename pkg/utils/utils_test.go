package utils

import (
	"math/big"
	"reflect"
	"testing"
)

func TestSet(t *testing.T) {
	tests := []struct {
		name string
		x    uint
		want []uint
	}{
		// TODO: Not sure whether [0] and [1] are valid sets but for the sake of completeness
		// here are the current results from Set
		{
			name: "zero set",
			x:    0,
			want: []uint{},
		},
		{
			name: "one",
			x:    1,
			want: []uint{0},
		},
		{
			name: "set",
			x:    10,
			want: []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}

	for _, tt := range tests {
		got := Set(tt.x)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: expected set:\n%v\ngot set:\n%v", tt.name, tt.want, got)
		}
	}
}

func TestConcat(t *testing.T) {
	tests := []struct {
		name string
		x, y *big.Int
		k    uint
		want *big.Int
	}{
		{
			name: "concat",
			x:    big.NewInt(4),
			y:    big.NewInt(4),
			k:    3,
			want: big.NewInt(36),
		},
	}

	for _, tt := range tests {
		got := Concat(tt.x, tt.y, tt.k)
		if got.Cmp(tt.want) != 0 {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestTrunc(t *testing.T) {
	tests := []struct {
		name    string
		x       *big.Int
		a, b, k uint
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
