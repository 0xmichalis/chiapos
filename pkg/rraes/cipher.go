// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rraes

import (
	"crypto/cipher"
	"strconv"
)

// The AES block size in bytes.
const BlockSize = 16

// A cipher is an instance of AES encryption using a particular key.
type aesCipher struct {
	enc []uint32
	dec []uint32
}

type KeySizeError int

func (k KeySizeError) Error() string {
	return "crypto/aes: invalid key size " + strconv.Itoa(int(k))
}

// NewCipher creates and returns a new cipher.Block.
// The key argument should be the AES key,
// 16 bytes long, to select AES-128.
func NewCipher(key []byte) (cipher.Block, error) {
	if k := len(key); k != 16 {
		return nil, KeySizeError(k)
	}
	return newCipher(key)
}

func (c *aesCipher) BlockSize() int { return BlockSize }

func (c *aesCipher) Encrypt(dst, src []byte) {
	panic("rraes: golang encryption not implemented")
}

func (c *aesCipher) Decrypt(dst, src []byte) {
	panic("rraes: decryption not implemented")
}
