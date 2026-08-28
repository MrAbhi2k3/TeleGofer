package crypto

import (
	"crypto/sha256"
)

// Role indicates whether the key derivation is for an outgoing client message
// or an incoming server message.
type Role int

const (
	ClientRole Role = 0 // x = 0
	ServerRole Role = 8 // x = 8
)

// KDF derives the 32-byte AES key and 32-byte AES IV for MTProto 2.0.
//
// Official MTProto 2.0 Math:
// sha256_a = SHA256(msg_key + auth_key[x : x+36])
// sha256_b = SHA256(auth_key[40+x : 40+x+36] + msg_key)
//
// aes_key = sha256_a[0:8] + sha256_b[8:24] + sha256_a[24:32] (8 + 16 + 8 = 32 bytes)
// aes_iv  = sha256_b[0:8] + sha256_a[8:24] + sha256_b[24:32] (8 + 16 + 8 = 32 bytes)
func KDF(authKey []byte, msgKey [16]byte, role Role) (key [32]byte, iv [32]byte) {
	x := int(role)

	// Pre-allocated stack buffers to avoid heap allocations
	var bufA [16 + 36]byte
	copy(bufA[0:16], msgKey[:])
	copy(bufA[16:52], authKey[x:x+36])
	shaA := sha256.Sum256(bufA[:])

	var bufB [36 + 16]byte
	copy(bufB[0:36], authKey[40+x:40+x+36])
	copy(bufB[36:52], msgKey[:])
	shaB := sha256.Sum256(bufB[:])

	// Construct aes_key (32 bytes)
	copy(key[0:8], shaA[0:8])
	copy(key[8:24], shaB[8:24])
	copy(key[24:32], shaA[24:32])

	// Construct aes_iv (32 bytes)
	copy(iv[0:8], shaB[0:8])
	copy(iv[8:24], shaA[8:24])
	copy(iv[24:32], shaB[24:32])

	return key, iv
}

// ComputeMsgKey computes the MTProto 2.0 16-byte msg_key from auth_key and plaintext.
//
// Math:
// msg_key_large = SHA256(auth_key[88+x : 88+x+32] + plaintext_including_padding)
// msg_key = msg_key_large[8 : 24]
func ComputeMsgKey(authKey []byte, plaintext []byte, role Role) [16]byte {
	x := int(role)

	h := sha256.New()
	h.Write(authKey[88+x : 88+x+32])
	h.Write(plaintext)
	sum := h.Sum(nil)

	var msgKey [16]byte
	copy(msgKey[:], sum[8:24])
	return msgKey
}
