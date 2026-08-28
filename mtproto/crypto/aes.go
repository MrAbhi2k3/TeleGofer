package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

var (
	ErrInvalidBlockSize = errors.New("crypto: data length must be a multiple of 16")
	ErrInvalidIVLength  = errors.New("crypto: IV length must be exactly 32 bytes for AES-IGE")
	ErrInvalidKeyLength = errors.New("crypto: key length must be 32 bytes for AES-256")
)

// IGE represents an AES-256 cipher operating in Infinite Garble Extension (IGE) mode,
// which is required by Telegram's MTProto specification.
type IGE struct {
	block cipher.Block
	iv    [32]byte
}

// NewIGE creates an IGE cipher using a 32-byte AES key and 32-byte IV.
func NewIGE(key, iv []byte) (*IGE, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}
	if len(iv) != 32 {
		return nil, ErrInvalidIVLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	c := &IGE{block: block}
	copy(c.iv[:], iv)
	return c, nil
}

// Encrypt encrypts src into dst using AES-256 in IGE mode.
// dst and src may point to the exact same memory slice for in-place encryption.
//
// Math:
// C_i = E_K(M_i ^ C_{i-1}) ^ M_{i-1}
// Where C_0 = iv[0:16], M_0 = iv[16:32]
func (c *IGE) Encrypt(dst, src []byte) error {
	n := len(src)
	if n%16 != 0 {
		return ErrInvalidBlockSize
	}
	if len(dst) < n {
		return errors.New("crypto: destination buffer is too small")
	}

	cPrev := c.iv[0:16]
	mPrev := c.iv[16:32]

	var t [16]byte

	for i := 0; i < n; i += 16 {
		mCurr := src[i : i+16]

		// t = mCurr ^ cPrev
		for j := 0; j < 16; j++ {
			t[j] = mCurr[j] ^ cPrev[j]
		}

		// Encrypt in-place into t
		c.block.Encrypt(t[:], t[:])

		// cCurr = t ^ mPrev
		for j := 0; j < 16; j++ {
			dst[i+j] = t[j] ^ mPrev[j]
		}

		cPrev = dst[i : i+16]
		mPrev = mCurr
	}

	return nil
}

// Decrypt decrypts src into dst using AES-256 in IGE mode.
// dst and src may point to the exact same memory slice for in-place decryption.
//
// Math:
// M_i = D_K(C_i ^ M_{i-1}) ^ C_{i-1}
// Where C_0 = iv[0:16], M_0 = iv[16:32]
func (c *IGE) Decrypt(dst, src []byte) error {
	n := len(src)
	if n%16 != 0 {
		return ErrInvalidBlockSize
	}
	if len(dst) < n {
		return errors.New("crypto: destination buffer is too small")
	}

	cPrev := c.iv[0:16]
	mPrev := c.iv[16:32]

	var t [16]byte

	for i := 0; i < n; i += 16 {
		cCurr := src[i : i+16]

		// t = cCurr ^ mPrev
		for j := 0; j < 16; j++ {
			t[j] = cCurr[j] ^ mPrev[j]
		}

		// Decrypt in-place into t
		c.block.Decrypt(t[:], t[:])

		// mCurr = t ^ cPrev
		for j := 0; j < 16; j++ {
			dst[i+j] = t[j] ^ cPrev[j]
		}

		mPrev = dst[i : i+16]
		cPrev = cCurr
	}

	return nil
}
