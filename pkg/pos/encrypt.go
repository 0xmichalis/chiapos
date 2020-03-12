package pos

import (
	"crypto/aes"
	"crypto/cipher"
	"math/big"

	"github.com/kargakis/chiapos/pkg/parameters"
	"github.com/kargakis/chiapos/pkg/serialize"
	"github.com/kargakis/chiapos/pkg/utils"
)

// At is a high-level hash function that calls AES on its inputs.
// c is meant to be created using the plot seed as a key.
func At(x, y *big.Int, k, t int, c cipher.Block) uint64 {
	param := new(big.Int).Lsh(big.NewInt(1), 128)

	// setup x low and high
	xLow := new(big.Int)
	xHigh := new(big.Int)
	xHigh.DivMod(x, param, xLow)

	// setup y low and high
	yLow := new(big.Int)
	yHigh := new(big.Int)
	yHigh.DivMod(y, param, yLow)

	// estimate collation size
	size := 2 * k * serialize.CollaSize(t)

	// main logic
	var cipherText [aes.BlockSize]byte
	switch {
	case 0 <= size && size <= 128:
		c.Encrypt(cipherText[:], utils.FillToBlock(utils.ConcatBig(k, x, y).Bytes()))

	case 129 <= size && size <= 256:
		c.Encrypt(cipherText[:], utils.FillToBlock(x.Bytes()))
		tmp := new(big.Int).SetBytes(cipherText[:])
		c.Encrypt(cipherText[:], utils.FillToBlock(tmp.Xor(tmp, y).Bytes()))

	case 257 <= size && size <= 384:
		var cipherConcat [aes.BlockSize]byte
		c.Encrypt(cipherConcat[:], utils.FillToBlock(utils.ConcatBig(k, xLow, yLow).Bytes()))
		ccBig := new(big.Int).SetBytes(cipherConcat[:])

		var cipherYHigh [aes.BlockSize]byte
		c.Encrypt(cipherYHigh[:], utils.FillToBlock(yHigh.Bytes()))
		cyBig := new(big.Int).SetBytes(cipherYHigh[:])

		var cipherXHigh [aes.BlockSize]byte
		c.Encrypt(cipherXHigh[:], utils.FillToBlock(xHigh.Bytes()))
		cxBig := new(big.Int).SetBytes(cipherXHigh[:])

		ccBig.Xor(ccBig, cyBig).Xor(ccBig, cxBig)
		c.Encrypt(cipherText[:], utils.FillToBlock(ccBig.Bytes()))

	case 385 <= size && size <= 512:
		var tmp [aes.BlockSize]byte
		c.Encrypt(tmp[:], utils.FillToBlock(xHigh.Bytes()))
		tmpBig := new(big.Int).SetBytes(tmp[:])
		c.Encrypt(tmp[:], utils.FillToBlock(tmpBig.Xor(tmpBig, xLow).Bytes()))
		tmpBig = new(big.Int).SetBytes(tmp[:])

		var cipherYHigh [aes.BlockSize]byte
		c.Encrypt(cipherYHigh[:], utils.FillToBlock(yHigh.Bytes()))
		cyBig := new(big.Int).SetBytes(cipherYHigh[:])

		c.Encrypt(cipherText[:], utils.FillToBlock(tmpBig.Xor(tmpBig, cyBig).Xor(tmpBig, yLow).Bytes()))
	}

	// need to return the most significant k+paramEXT bits
	res := new(big.Int).SetBytes(cipherText[:])
	return utils.Trunc(res, 0, k+parameters.ParamEXT, res.BitLen()).Uint64()
}
