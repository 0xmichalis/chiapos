package pos

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"math"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/utils"
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

// Calculate accepts a number and calculates a batch of
// 2^(k+kExtraBits)-bit outputs.
func (f *F1) Calculate(x uint64) [][]byte {
	counter := (x * uint64(f.k)) / kBlockSizeBits

	cipherBytes := mybits.ToBytes(f.k * kBlockSizeBits)
	ciphertext := make([]byte, cipherBytes)
	var index, start, end int
	for cipherBytes > end {
		counterBytes := mybits.Uint64ToBytes(counter, f.k)
		start = index * aes.BlockSize % (cipherBytes + 1)
		end = ((index + 1) * aes.BlockSize) % (cipherBytes + 1)
		f.key.Encrypt(ciphertext[start:end], utils.FillToBlock(counterBytes))
		counter++
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
				left = append(left, lb)
				right = append(right, rb)
				leftSize = 8 - (f.k - leftSize)
			} else {
				left = append(left, c)
				leftSize = 0
			}
			outputs = append(outputs, left)
			extended := mybits.Uint64ToBytes(x+xIndex%parameters.ParamM, parameters.ParamEXT)
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
