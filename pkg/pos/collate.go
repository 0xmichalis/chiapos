package pos

import (
	"math/big"

	"github.com/kargakis/gochia/pkg/utils"
)

// Collate collates left and right inputs into outputs for the next table.
func Collate(t, k int, l, r *big.Int) (*big.Int, error) {
	switch t {
	case 2:
		return utils.ConcatBig(k, l, r), nil

	case 3:
		return utils.ConcatBig(2*k, l, r), nil

	case 4:
		return l.Xor(l, r), nil

	case 5:
		// TODO: When bytes are serialized to a primitive such as int or big.Int
		// the most significant bits can be empty so less then the expected size
		// which is kind of expected when dealing with random numbers.
		//if l.BitLen()%4 != 0 {
		//	return nil, fmt.Errorf("invalid bit length for output %d, expected bit_len%%4==0", l.BitLen())
		//}
		l.Xor(l, r)
		return utils.Trunc(l, 0, l.BitLen()*3/4, l.BitLen()), nil

	case 6:
		// TODO: When bytes are serialized to a primitive such as int or big.Int
		// the most significant bits can be empty so less then the expected size.
		// which is kind of expected when dealing with random numbers.
		//if l.BitLen()%3 != 0 {
		//	return nil, fmt.Errorf("invalid bit length for output %d, expected bit_len%%3==0", l.BitLen())
		//}
		l.Xor(l, r)
		return utils.Trunc(l, 0, l.BitLen()*2/3, l.BitLen()), nil
	}
	return nil, nil
}
