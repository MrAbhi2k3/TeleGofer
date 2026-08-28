package crypto

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
)

// AuthKey represents a 2048-bit (256-byte) shared authorization key created
// between client and server via Diffie-Hellman exchange.
type AuthKey struct {
	Value [256]byte
	ID    uint64 // Lower 64 bits of SHA1(auth_key), transmitted in unencrypted header
}

// NewAuthKey constructs an AuthKey from a 256-byte slice and calculates
// its 64-bit key identifier per MTProto specification.
func NewAuthKey(data []byte) (*AuthKey, error) {
	if len(data) != 256 {
		return nil, errors.New("crypto: auth_key must be exactly 256 bytes")
	}

	ak := &AuthKey{}
	copy(ak.Value[:], data)

	// Auth key ID = lower 64 bits of SHA1(auth_key)
	h := sha1.Sum(ak.Value[:])
	ak.ID = binary.LittleEndian.Uint64(h[12:20])

	return ak, nil
}
