package pos

import "testing"

func TestGetLeftAndRight(t *testing.T) {
	tests := []struct {
		name      string
		c         byte
		remaining int

		expLeft  byte
		expRight byte
	}{
		{
			name:      "normal",
			c:         0b01010101,
			remaining: 5,
			expLeft:   0b01010,
			expRight:  0b101,
		},
		{
			name:      "edge",
			c:         0b11010101,
			remaining: 1,
			expLeft:   0b1,
			expRight:  0b1010101,
		},
	}

	for _, test := range tests {
		left, right := getLeftAndRight(test.c, test.remaining)
		if left != test.expLeft {
			t.Fatalf("%s: expected left byte: %08b, got: %08b", test.name, test.expLeft, left)
		}
		if right != test.expRight {
			t.Fatalf("%s: expected right byte: %08b, got: %08b", test.name, test.expRight, right)
		}
	}
}
