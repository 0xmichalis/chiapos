package pos

import (
	"crypto/cipher"
	"math/big"

	"github.com/kargakis/gochia/pkg/utils"
)

// At is a high-level hash function that calls AES on its inputs.
// c is meant to be created using the plot seed as a key.
func At(x, y, k uint64, t int, c cipher.Block) (*uint64, error) {
	param := big.NewInt(1).Rsh(big.NewInt(1), 128)

	// setup x low and high
	xBig := new(big.Int).SetUint64(x)
	xLow := new(big.Int)
	xHigh := new(big.Int)
	xHigh.DivMod(xBig, param, xLow)

	// setup y low and high
	yBig := new(big.Int).SetUint64(y)
	yLow := new(big.Int)
	yHigh := new(big.Int)
	yHigh.DivMod(yBig, param, yLow)

	// setup size
	collaSize, err := CollaSize(t)
	if err != nil {
		return nil, err
	}
	size := 2 * int(k) * *collaSize

	// main logic
	var cipherText []byte
	switch {
	case 0 <= size && size <= 128:
		plaintext := utils.Concat(k, x, y)
		c.Encrypt(cipherText, plaintext.Bytes())

	case 129 <= size && size <= 256:
		c.Encrypt(cipherText, xBig.Bytes())
		tmp := new(big.Int).SetBytes(cipherText)
		c.Encrypt(cipherText, tmp.Xor(tmp, yBig).Bytes())

	case 257 <= size && size <= 384:
		var cipherConcat []byte
		c.Encrypt(cipherConcat, utils.Concat(k, xLow.Uint64(), yLow.Uint64()).Bytes())
		ccBig := new(big.Int).SetBytes(cipherConcat)

		var cipherYHigh []byte
		c.Encrypt(cipherYHigh, yHigh.Bytes())
		cyBig := new(big.Int).SetBytes(cipherYHigh)

		var cipherXHigh []byte
		c.Encrypt(cipherXHigh, xHigh.Bytes())
		cxBig := new(big.Int).SetBytes(cipherXHigh)

		ccBig.Xor(ccBig, cyBig).Xor(ccBig, cxBig)
		c.Encrypt(cipherText, ccBig.Bytes())

	case 385 <= size && size <= 512:
		var tmp []byte
		c.Encrypt(tmp, xHigh.Bytes())
		tmpBig := new(big.Int).SetBytes(tmp)
		c.Encrypt(tmp, tmpBig.Xor(tmpBig, xLow).Bytes())
		tmpBig = new(big.Int).SetBytes(tmp)

		var cipherYHigh []byte
		c.Encrypt(cipherYHigh, yHigh.Bytes())
		cyBig := new(big.Int).SetBytes(cipherYHigh)

		c.Encrypt(cipherText, tmpBig.Xor(tmpBig, cyBig).Xor(tmpBig, yLow).Bytes())
	}

	// need to return the most significant k+paramEXT bits
	res := new(big.Int).SetBytes(cipherText)
	r := utils.Trunc(res, 0, k+5, k).Uint64()
	return &r, nil
}
