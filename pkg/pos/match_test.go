package pos

import "testing"

// TODO: Compare with the efficient impl
func TestMatchNaive(t *testing.T) {
	tests := []struct {
		left  uint64
		right uint64

		expected bool
	}{}

	for i, test := range tests {
		got := matchNaive(test.left, test.right)
		if got != test.expected {
			t.Fatalf("%d: expected %t, got %t", i, test.expected, got)
		}
	}
}
