// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rraes

import (
	"crypto/cipher"
)

// defined in asm_*.s

//go:noescape
func encryptBlockAsm(nr int, xk *uint32, dst, src *byte)

//go:noescape
func expandKeyAsm(nr int, key *byte, enc *uint32, dec *uint32)

type aesCipherAsm struct {
	aesCipher
}

func newCipher(key []byte) (cipher.Block, error) {
	n := len(key) + 28
	c := aesCipherAsm{aesCipher{make([]uint32, n), make([]uint32, n)}}

	expandKeyAsm(10, &key[0], &c.enc[0], &c.dec[0])
	return &c, nil
}

func (c *aesCipherAsm) BlockSize() int { return BlockSize }

func (c *aesCipherAsm) Encrypt(dst, src []byte) {
	if len(src) < BlockSize {
		panic("rraes: input not full block")
	}
	if len(dst) < BlockSize {
		panic("rraes: output not full block")
	}
	if InexactOverlap(dst[:BlockSize], src[:BlockSize]) {
		panic("rraes: invalid buffer overlap")
	}
	encryptBlockAsm(len(c.enc)/4-1, &c.enc[0], &dst[0], &src[0])
}

func (c *aesCipherAsm) Decrypt(dst, src []byte) {
	panic("rraes: decryption not implemented")
}

// expandKey is used by BenchmarkExpand to ensure that the asm implementation
// of key expansion is used for the benchmark when it is available.
func expandKey(key []byte, enc, dec []uint32) {
	expandKeyAsm(10, &key[0], &enc[0], &dec[0])
}
