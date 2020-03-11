package pos

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"math"
	"math/big"

	"github.com/kargakis/chiapos/pkg/utils"

	"github.com/kargakis/chiapos/pkg/parameters"
	mybits "github.com/kargakis/chiapos/pkg/utils/bits"
)

const (
	// AES block size
	kBlockSizeBits = aes.BlockSize * 8
)

type F1 struct {
	k   int
	key cipher.Block
}

func NewF1(k int, key []byte) (*F1, error) {
	if k < parameters.KMinPlotSize || k > parameters.KMaxPlotSize {
		return nil, fmt.Errorf("invalid k: %d, valid range: %d - %d", k, parameters.KMinPlotSize, parameters.KMaxPlotSize)
	}

	f1 := &F1{
		k: k,
	}

	aesKey := make([]byte, 32)
	copy(aesKey[:], key)

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	f1.key = block

	return f1, nil
}

func (f *F1) CalculateOne(x uint64) uint64 {
	q, r := new(big.Int).DivMod(new(big.Int).SetUint64(x*uint64(f.k)), big.NewInt(kBlockSizeBits), new(big.Int))
	// fmt.Printf("q=%d, r=%d, x=%d, k=%d\n", q.Uint64(), r.Uint64(), x, f.k)

	var qCipher [16]byte
	data := utils.FillToBlock(q.Bytes())
	f.key.Encrypt(qCipher[:], data)
	res := new(big.Int).SetBytes(qCipher[:])

	if int(r.Uint64())+f.k <= kBlockSizeBits {
		res = utils.Trunc(res, int(r.Uint64()), int(r.Uint64())+f.k, kBlockSizeBits)
	} else {
		part1 := utils.Trunc(res, int(r.Uint64()), kBlockSizeBits, kBlockSizeBits)
		var q1Cipher [16]byte
		data := utils.FillToBlock(q.Add(q, big.NewInt(1)).Bytes())
		f.key.Encrypt(q1Cipher[:], data)
		part2 := new(big.Int).SetBytes(q1Cipher[:])
		part2 = utils.Trunc(part2, 0, int(r.Uint64())+f.k-kBlockSizeBits, kBlockSizeBits)
		res = utils.Concat(uint64(part2.BitLen()), part1.Uint64(), part2.Uint64())
	}

	f1x := utils.ConcatExtended(res.Uint64(), x)
	// fmt.Printf("Calculated f1(x)=%d for x=%d\n", f1x, x)
	return f1x
}

// Calculate accepts a number and calculates a batch of
// 2^(k+kExtraBits)-bit outputs.
// TODO: Currently this impl is generating a lot of matches
// in comparison to the naive approach (CalculateOne). Figure
// out why.
func (f *F1) Calculate(x uint64) [][]byte {
	cipherBytes := mybits.ToBytes(f.k * kBlockSizeBits)
	ciphertext := make([]byte, cipherBytes)
	var index, start, end int

	for cipherBytes > end {
		start = index * aes.BlockSize % (cipherBytes + 1)
		end = ((index + 1) * aes.BlockSize) % (cipherBytes + 1)
		counterBytes := mybits.Uint64ToBytes(x, kBlockSizeBits)
		f.key.Encrypt(ciphertext[start:end], counterBytes)
		x++
		index++
	}

	var (
		outputs     [][]byte
		left, right []byte
		xIndex      uint64
		leftSize    int
		needsTrunc  = mybits.ToBytes(f.k) != f.k/8
	)

	// slice the ciphertext properly to get back all the f(x)s
	for i, c := range ciphertext {
		if !containsTwoKs(i, f.k) {
			left = append(left, c)
			leftSize += 8
		} else {
			if needsTrunc {
				lb, rb := getLeftAndRight(c, f.k-leftSize)
				// TODO: This append is wrong, what is needed here for the
				// left byte is to shift all other left bytes f.k-leftSize bits
				// to the left, handle overflows, then add lb.
				left = append(left, lb)
				right = append(right, rb)
				// the remaining bits (right) are assigned to leftSize
				// since right will become left in the next iteration.
				leftSize = 8 - (f.k - leftSize)
			} else {
				left = append(left, c)
				leftSize = 0
			}
			outputs = append(outputs, left)
			extended := mybits.Uint64ToBytes((x+xIndex)%parameters.ParamM, parameters.ParamEXT)
			// TODO: Similar to the lb append above, this is also wrong.
			outputs[xIndex] = append(outputs[xIndex], extended...)
			xIndex++
			// clean up buffers
			left = right
			right = nil
		}
	}

	return outputs
}

func containsTwoKs(index, k int) bool {
	return index*8/k != (index+1)*8/k
}

func getLeftAndRight(c byte, remainingLeftSize int) (byte, byte) {
	var mask int
	if remainingLeftSize == 1 {
		// the mask will overflow so we need to handle this case manually
		mask = 7
	} else {
		mask = 8 - remainingLeftSize + 1
	}
	m := byte(math.Pow(2, float64(mask)))

	left := c >> (8 - remainingLeftSize)
	right := c % m
	return left, right
}
