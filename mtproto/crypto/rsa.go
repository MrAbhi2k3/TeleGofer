package crypto

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"math/big"
)

// RSAPublicKey represents an RSA public key used during MTProto DH key exchange.
type RSAPublicKey struct {
	N           *big.Int // Modulus (2048-bit)
	E           *big.Int // Exponent (typically 65537)
	Fingerprint int64    // 64-bit fingerprint
}

// ComputeFingerprint computes the lower 64-bit SHA1 fingerprint of a serialized RSA public key.
func ComputeFingerprint(n, e *big.Int) int64 {
	// Standard Telegram fingerprint: SHA1 of (string(n) + string(e))
	nBytes := n.Bytes()
	eBytes := e.Bytes()

	var buf []byte
	// Length-prefixed serialization
	buf = append(buf, encodeTLBytes(nBytes)...)
	buf = append(buf, encodeTLBytes(eBytes)...)

	h := sha1.Sum(buf)
	return int64(binary.LittleEndian.Uint64(h[12:20]))
}

func encodeTLBytes(b []byte) []byte {
	n := len(b)
	var out []byte
	if n <= 253 {
		out = append(out, byte(n))
		out = append(out, b...)
		pad := (4 - ((1 + n) % 4)) % 4
		for i := 0; i < pad; i++ {
			out = append(out, 0)
		}
	} else {
		out = append(out, 0xfe, byte(n), byte(n>>8), byte(n>>16))
		out = append(out, b...)
		pad := (4 - (n % 4)) % 4
		for i := 0; i < pad; i++ {
			out = append(out, 0)
		}
	}
	return out
}

// EncryptRSA performs raw RSA modular exponentiation c = m^e mod N with MTProto random padding.
func (k *RSAPublicKey) Encrypt(data []byte) ([]byte, error) {
	// Telegram MTProto RSA encryption requires 255 bytes of padded data:
	// data_with_hash = SHA1(data) + data + random_padding up to 255 bytes
	h := sha1.Sum(data)
	contentLen := len(h) + len(data)
	if contentLen > 255 {
		return nil, errors.New("crypto: data too large for 2048-bit RSA encryption")
	}

	padded := make([]byte, 255)
	copy(padded[0:20], h[:])
	copy(padded[20:contentLen], data)

	if 255-contentLen > 0 {
		if _, err := rand.Read(padded[contentLen:]); err != nil {
			return nil, err
		}
	}

	m := new(big.Int).SetBytes(padded)
	c := new(big.Int).Exp(m, k.E, k.N)

	// Result must be 256 bytes
	out := c.Bytes()
	if len(out) < 256 {
		fixed := make([]byte, 256)
		copy(fixed[256-len(out):], out)
		return fixed, nil
	}
	return out, nil
}
