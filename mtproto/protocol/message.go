package protocol

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/mrabhi2k3/telegofer/mtproto/crypto"
)

var (
	ErrPacketTooShort   = errors.New("protocol: packet is too short")
	ErrAuthKeyMismatch  = errors.New("protocol: auth_key_id does not match")
	ErrMsgKeyMismatch   = errors.New("protocol: message key verification failed")
	ErrSessionMismatch  = errors.New("protocol: session ID does not match")
	ErrInvalidMsgLength = errors.New("protocol: invalid inner message length")
)

// PackUnencrypted encodes an unencrypted MTProto message (used solely during DH auth key creation).
//
// Format:
// auth_key_id (8 bytes, 0x0000000000000000)
// msg_id      (8 bytes)
// msg_len     (4 bytes, length of payload)
// payload     (N bytes)
func PackUnencrypted(msgID int64, payload []byte) []byte {
	out := make([]byte, 8+8+4+len(payload))
	// auth_key_id is 0
	binary.LittleEndian.PutUint64(out[8:16], uint64(msgID))
	binary.LittleEndian.PutUint32(out[16:20], uint32(len(payload)))
	copy(out[20:], payload)
	return out
}

// UnpackUnencrypted decodes an unencrypted MTProto message and returns the msgID and payload.
func UnpackUnencrypted(data []byte) (int64, []byte, error) {
	if len(data) < 20 {
		return 0, nil, ErrPacketTooShort
	}

	authKeyID := binary.LittleEndian.Uint64(data[0:8])
	if authKeyID != 0 {
		return 0, nil, fmt.Errorf("protocol: expected unencrypted auth_key_id 0, got 0x%016x", authKeyID)
	}

	msgID := int64(binary.LittleEndian.Uint64(data[8:16]))
	msgLen := int(binary.LittleEndian.Uint32(data[16:20]))

	if len(data) < 20+msgLen {
		return 0, nil, ErrPacketTooShort
	}

	return msgID, data[20 : 20+msgLen], nil
}

func PackEncrypted(authKey *crypto.AuthKey, serverSalt int64, sessionID int64, msgID int64, seqNo int32, payload []byte) ([]byte, error) {
	headerLen := 8 + 8 + 8 + 4 + 4 // 32 bytes
	contentLen := headerLen + len(payload)

	// In MTProto 2.0, padding is 12..1024 bytes and contentLen + padLen must be a multiple of 16
	padLen := (16 - (contentLen % 16))
	if padLen < 12 {
		padLen += 16
	}

	totalInner := contentLen + padLen
	inner := make([]byte, totalInner)

	binary.LittleEndian.PutUint64(inner[0:8], uint64(serverSalt))
	binary.LittleEndian.PutUint64(inner[8:16], uint64(sessionID))
	binary.LittleEndian.PutUint64(inner[16:24], uint64(msgID))
	binary.LittleEndian.PutUint32(inner[24:28], uint32(seqNo))
	binary.LittleEndian.PutUint32(inner[28:32], uint32(len(payload)))
	copy(inner[32:32+len(payload)], payload)

	// Fill padding with cryptographically secure random bytes
	if _, err := rand.Read(inner[contentLen:]); err != nil {
		return nil, fmt.Errorf("protocol: crypto/rand error: %w", err)
	}

	// 1. Calculate msg_key = middle 128 bits of SHA256(auth_key[88:120] + inner)
	msgKey := crypto.ComputeMsgKey(authKey.Value[:], inner, crypto.ClientRole)

	// 2. Derive AES key and IV using KDF
	aesKey, aesIV := crypto.KDF(authKey.Value[:], msgKey, crypto.ClientRole)

	// 3. Encrypt inner data in-place using AES-256 in IGE mode
	cipher, err := crypto.NewIGE(aesKey[:], aesIV[:])
	if err != nil {
		return nil, err
	}
	if err := cipher.Encrypt(inner, inner); err != nil {
		return nil, err
	}

	// 4. Assemble outer packet: auth_key_id (8 bytes) + msg_key (16 bytes) + encrypted_data
	outer := make([]byte, 8+16+len(inner))
	binary.LittleEndian.PutUint64(outer[0:8], authKey.ID)
	copy(outer[8:24], msgKey[:])
	copy(outer[24:], inner)

	return outer, nil
}

// UnpackEncrypted decrypts and validates an incoming encrypted MTProto 2.0 packet from the server.
func UnpackEncrypted(authKey *crypto.AuthKey, expectedSessionID int64, data []byte) (serverSalt int64, msgID int64, seqNo int32, payload []byte, err error) {
	if len(data) < 8+16+32 { // Minimum valid packet size
		return 0, 0, 0, nil, ErrPacketTooShort
	}

	authKeyID := binary.LittleEndian.Uint64(data[0:8])
	if authKeyID != authKey.ID {
		return 0, 0, 0, nil, ErrAuthKeyMismatch
	}

	var msgKey [16]byte
	copy(msgKey[:], data[8:24])

	encrypted := data[24:]
	if len(encrypted)%16 != 0 {
		return 0, 0, 0, nil, ErrPacketTooShort
	}

	// 1. Derive AES key and IV using ServerRole (x = 8)
	aesKey, aesIV := crypto.KDF(authKey.Value[:], msgKey, crypto.ServerRole)

	cipher, err := crypto.NewIGE(aesKey[:], aesIV[:])
	if err != nil {
		return 0, 0, 0, nil, err
	}

	inner := make([]byte, len(encrypted))
	if err := cipher.Decrypt(inner, encrypted); err != nil {
		return 0, 0, 0, nil, err
	}

	// 2. Verify msg_key matches SHA256 of decrypted inner plaintext
	expectedMsgKey := crypto.ComputeMsgKey(authKey.Value[:], inner, crypto.ServerRole)
	if msgKey != expectedMsgKey {
		return 0, 0, 0, nil, ErrMsgKeyMismatch
	}

	serverSalt = int64(binary.LittleEndian.Uint64(inner[0:8]))
	sessionID := int64(binary.LittleEndian.Uint64(inner[8:16]))
	if expectedSessionID != 0 && sessionID != expectedSessionID {
		return 0, 0, 0, nil, ErrSessionMismatch
	}

	msgID = int64(binary.LittleEndian.Uint64(inner[16:24]))
	seqNo = int32(binary.LittleEndian.Uint32(inner[24:28]))
	msgLen := int(binary.LittleEndian.Uint32(inner[28:32]))

	if msgLen < 0 || 32+msgLen > len(inner) {
		return 0, 0, 0, nil, ErrInvalidMsgLength
	}

	payload = make([]byte, msgLen)
	copy(payload, inner[32:32+msgLen])

	return serverSalt, msgID, seqNo, payload, nil
}
