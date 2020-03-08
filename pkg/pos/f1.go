package pos

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/utils"
	"github.com/kargakis/chiapos/pkg/utils/bits"
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

	cipherBytes := bits.ToBytes(f.k * kBlockSizeBits)
	ciphertext := make([]byte, cipherBytes)
	var index, start, end int
	for cipherBytes > end {
		counterBytes := bits.Uint64ToBytes(counter, f.k)
		start = index * aes.BlockSize % (cipherBytes + 1)
		end = ((index + 1) * aes.BlockSize) % (cipherBytes + 1)
		f.key.Encrypt(ciphertext[start:end], utils.FillToBlock(counterBytes))
		counter++
		index++
	}

	var outputs [][]byte
	kBytes := bits.ToBytes(f.k)
	needsTrunc := kBytes != f.k*8
	tmp := make([]byte, kBytes)
	// slice the ciphertext properly to get back all the f(x)s
	var xIndex uint64
	for i, c := range ciphertext {
		if (i+1)%kBytes != 0 {
			tmp[i%kBytes] = c
		} else {
			if needsTrunc {
				tmp[i%kBytes] = c << (8 - (f.k % 8))
			} else {
				tmp[i%kBytes] = c
			}
			outputs = append(outputs, tmp)
			extended := bits.Uint64ToBytes(x+xIndex%parameters.ParamM, parameters.ParamEXT)
			outputs[xIndex] = append(outputs[xIndex], extended...)
			xIndex++
			// clean up buffer
			tmp = make([]byte, kBytes)
		}
	}

	return outputs
}
