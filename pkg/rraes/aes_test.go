// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rraes

import (
	"testing"
)

// See const.go for overview of math here.

// Test that powx is initialized correctly.
// (Can adapt this code to generate it too.)
func TestPowx(t *testing.T) {
	p := 1
	for i := 0; i < len(powx); i++ {
		if powx[i] != byte(p) {
			t.Errorf("powx[%d] = %#x, want %#x", i, powx[i], p)
		}
		p <<= 1
		if p&0x100 != 0 {
			p ^= poly
		}
	}
}

// Multiply b and c as GF(2) polynomials modulo poly
func mul(b, c uint32) uint32 {
	i := b
	j := c
	s := uint32(0)
	for k := uint32(1); k < 0x100 && j != 0; k <<= 1 {
		// Invariant: k == 1<<n, i == b * xⁿ

		if j&k != 0 {
			// s += i in GF(2); xor in binary
			s ^= i
			j ^= k // turn off bit to end loop early
		}

		// i *= x in GF(2) modulo the polynomial
		i <<= 1
		if i&0x100 != 0 {
			i ^= poly
		}
	}
	return s
}

// Test all mul inputs against bit-by-bit n² algorithm.
func TestMul(t *testing.T) {
	for i := uint32(0); i < 256; i++ {
		for j := uint32(0); j < 256; j++ {
			// Multiply i, j bit by bit.
			s := uint8(0)
			for k := uint(0); k < 8; k++ {
				for l := uint(0); l < 8; l++ {
					if i&(1<<k) != 0 && j&(1<<l) != 0 {
						s ^= powx[k+l]
					}
				}
			}
			if x := mul(i, j); x != uint32(s) {
				t.Fatalf("mul(%#x, %#x) = %#x, want %#x", i, j, x, s)
			}
		}
	}
}

// Check that S-boxes are inverses of each other.
// They have more structure that we could test,
// but if this sanity check passes, we'll assume
// the cut and paste from the FIPS PDF worked.
func TestSboxes(t *testing.T) {
	for i := 0; i < 256; i++ {
		if j := sbox0[sbox1[i]]; j != byte(i) {
			t.Errorf("sbox0[sbox1[%#x]] = %#x", i, j)
		}
		if j := sbox1[sbox0[i]]; j != byte(i) {
			t.Errorf("sbox1[sbox0[%#x]] = %#x", i, j)
		}
	}
}

// Test that encryption tables are correct.
// (Can adapt this code to generate them too.)
func TestTe(t *testing.T) {
	for i := 0; i < 256; i++ {
		s := uint32(sbox0[i])
		s2 := mul(s, 2)
		s3 := mul(s, 3)
		w := s2<<24 | s<<16 | s<<8 | s3
		te := [][256]uint32{te0, te1, te2, te3}
		for j := 0; j < 4; j++ {
			if x := te[j][i]; x != w {
				t.Fatalf("te[%d][%d] = %#x, want %#x", j, i, x, w)
			}
			w = w<<24 | w>>8
		}
	}
}

// Test that decryption tables are correct.
// (Can adapt this code to generate them too.)
func TestTd(t *testing.T) {
	for i := 0; i < 256; i++ {
		s := uint32(sbox1[i])
		s9 := mul(s, 0x9)
		sb := mul(s, 0xb)
		sd := mul(s, 0xd)
		se := mul(s, 0xe)
		w := se<<24 | s9<<16 | sd<<8 | sb
		td := [][256]uint32{td0, td1, td2, td3}
		for j := 0; j < 4; j++ {
			if x := td[j][i]; x != w {
				t.Fatalf("td[%d][%d] = %#x, want %#x", j, i, x, w)
			}
			w = w<<24 | w>>8
		}
	}
}

// Appendix B, C of FIPS 197: Cipher examples, Example vectors.
type CryptTest struct {
	key []byte
	in  []byte
	out []byte
}

var encryptTests = []CryptTest{
	{
		[]byte{0x2b, 0x7e, 0x15, 0x16, 0x28, 0xae, 0xd2, 0xa6, 0xab, 0xf7, 0x15, 0x88, 0x09, 0xcf, 0x4f, 0x3c},
		[]byte{0x32, 0x43, 0xf6, 0xa8, 0x88, 0x5a, 0x30, 0x8d, 0x31, 0x31, 0x98, 0xa2, 0xe0, 0x37, 0x07, 0x34},
		[]byte{0xaa, 0x8f, 0x5f, 0x03, 0x61, 0xdd, 0xe3, 0xef, 0x82, 0xd2, 0x4a, 0xd2, 0x68, 0x32, 0x46, 0x9a},
	},
	{
		[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		[]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		[]byte{0x49, 0x15, 0x59, 0x8f, 0x55, 0xe5, 0xd7, 0xa0, 0xda, 0xca, 0x94, 0xfa, 0x1f, 0x0a, 0x63, 0xf7},
	},
}

// Test Cipher Encrypt method against FIPS 197 examples.
func TestCipherEncrypt(t *testing.T) {
	for i, tt := range encryptTests {
		c, err := NewCipher(tt.key)
		if err != nil {
			t.Errorf("NewCipher(%d bytes) = %s", len(tt.key), err)
			continue
		}
		out := make([]byte, len(tt.in))
		c.Encrypt(out, tt.in)
		for j, v := range out {
			if v != tt.out[j] {
				t.Errorf("Cipher.Encrypt %d: out[%d] = %#x, want %#x", i, j, v, tt.out[j])
				break
			}
		}
	}
}

// Test short input/output.
// Assembly used to not notice.
// See issue 7928.
func TestShortBlocks(t *testing.T) {
	bytes := func(n int) []byte { return make([]byte, n) }

	c, _ := NewCipher(bytes(16))

	mustPanic(t, "rraes: input not full block", func() { c.Encrypt(bytes(1), bytes(1)) })
	mustPanic(t, "rraes: decryption not implemented", func() { c.Decrypt(bytes(1), bytes(1)) })
	mustPanic(t, "rraes: input not full block", func() { c.Encrypt(bytes(100), bytes(1)) })
	mustPanic(t, "rraes: decryption not implemented", func() { c.Decrypt(bytes(100), bytes(1)) })
	mustPanic(t, "rraes: output not full block", func() { c.Encrypt(bytes(1), bytes(100)) })
	mustPanic(t, "rraes: decryption not implemented", func() { c.Decrypt(bytes(1), bytes(100)) })
}

func mustPanic(t *testing.T, msg string, f func()) {
	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("function did not panic, wanted %q", msg)
		} else if err != msg {
			t.Errorf("got panic %q, wanted %q", err, msg)
		}
	}()
	f()
}

func BenchmarkEncrypt(b *testing.B) {
	tt := encryptTests[0]
	c, err := NewCipher(tt.key)
	if err != nil {
		b.Fatal("NewCipher:", err)
	}
	out := make([]byte, len(tt.in))
	b.SetBytes(int64(len(out)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Encrypt(out, tt.in)
	}
}

func BenchmarkExpand(b *testing.B) {
	tt := encryptTests[0]
	n := len(tt.key) + 28
	c := &aesCipher{make([]uint32, n), make([]uint32, n)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expandKey(tt.key, c.enc, c.dec)
	}
}
