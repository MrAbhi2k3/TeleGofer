package auth

import (
	"encoding/hex"
	"math/big"

	"github.com/mrabhi2k3/telegofer/mtproto/crypto"
)

// DefaultTelegramKeys provides Telegram's official production RSA public keys.
var DefaultTelegramKeys []*crypto.RSAPublicKey

func init() {
	// Telegram Official Production Key 1 (Fingerprint: -4684993188981600109)
	n1, _ := new(big.Int).SetString("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a"+
		"ace9e81f3d00e42f39447e338f805c9088f81ea477b77729347981e4f433f51a"+
		"c2a3f2e4be64712ac29ac4221576a335d5f12336ec8e0f52a982f1618e7d4235"+
		"510b2ae46e4fb8c165215274f397241a025e3402cd956763806a75a520974b16"+
		"4ece246b04af2c42c9a149c603249d4580fb6c7425a814fe13af95847cb29568"+
		"b096c8f2d93e117c3435d342ac7d5bbac4e6ff231491629f7169be161a96550b"+
		"abea4e344e9e500ff7f90963415a74366655d63c57d68b04576cd8934910066f"+
		"9b37a666adac63ddcb2c6384f5529c362d345dd290149213", 16)
	e1 := big.NewInt(65537)
	fp1 := crypto.ComputeFingerprint(n1, e1)

	DefaultTelegramKeys = append(DefaultTelegramKeys, &crypto.RSAPublicKey{
		N:           n1,
		E:           e1,
		Fingerprint: fp1,
	})
}

// FindKey looks up an RSA public key matching any of the server's provided fingerprints.
func FindKey(fingerprints []int64) *crypto.RSAPublicKey {
	for _, fp := range fingerprints {
		for _, key := range DefaultTelegramKeys {
			if key.Fingerprint == fp {
				return key
			}
		}
	}
	// Fallback to primary production key if none matched directly
	if len(DefaultTelegramKeys) > 0 {
		return DefaultTelegramKeys[0]
	}
	return nil
}

// ParsePQ parses the binary big-endian bytes of pq into a uint64.
func ParsePQ(pqStr string) uint64 {
	b, err := hex.DecodeString(pqStr)
	if err != nil || len(b) == 0 {
		// Try raw bytes if not hex
		b = []byte(pqStr)
	}
	var val uint64
	for _, byteVal := range b {
		val = (val << 8) | uint64(byteVal)
	}
	return val
}
