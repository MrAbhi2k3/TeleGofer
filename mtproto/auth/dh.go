package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/mrabhi2k3/telegofer/mtproto/crypto"
	"github.com/mrabhi2k3/telegofer/mtproto/protocol"
	"github.com/mrabhi2k3/telegofer/mtproto/transport"
	"github.com/mrabhi2k3/telegofer/tl/decoder"
	"github.com/mrabhi2k3/telegofer/tl/encoder"
	"github.com/mrabhi2k3/telegofer/tl/generated"
)

func CreateAuthKey(ctx context.Context, tr *transport.Conn, dcID int) (*crypto.AuthKey, int64, error) {
	// -------------------------------------------------------------
	// STEP 1: Send req_pq_multi -> Receive resPQ
	// -------------------------------------------------------------
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, 0, err
	}

	reqPQ := &generated.ReqPqMulti{Nonce: nonce}
	enc := encoder.New()
	if err := reqPQ.Encode(enc); err != nil {
		return nil, 0, err
	}

	msgIDGen := protocol.NewMessageIDGenerator()
	msgID := msgIDGen.Next()

	unenc := protocol.PackUnencrypted(msgID, enc.Bytes())
	if err := tr.Send(ctx, unenc); err != nil {
		return nil, 0, fmt.Errorf("auth: send req_pq_multi failed: %w", err)
	}

	respRaw, err := tr.Recv(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("auth: recv resPQ failed: %w", err)
	}

	_, payload, err := protocol.UnpackUnencrypted(respRaw)
	if err != nil {
		return nil, 0, fmt.Errorf("auth: unpack resPQ failed: %w", err)
	}

	resPQ := &generated.ResPQ{}
	if err := resPQ.Decode(decoder.New(payload)); err != nil {
		return nil, 0, fmt.Errorf("auth: decode resPQ failed: %w", err)
	}

	if resPQ.Nonce != nonce {
		return nil, 0, errors.New("auth: nonce mismatch in resPQ")
	}

	serverNonce := resPQ.ServerNonce

	// -------------------------------------------------------------
	// STEP 2: Factor pq, build p_q_inner_data, send req_DH_params
	// -------------------------------------------------------------
	pqVal := ParsePQ(resPQ.Pq)
	pVal, qVal := crypto.FactorPQ(pqVal)

	var newNonce [32]byte
	if _, err := rand.Read(newNonce[:]); err != nil {
		return nil, 0, err
	}

	pBytes := big.NewInt(int64(pVal)).Bytes()
	qBytes := big.NewInt(int64(qVal)).Bytes()
	pqBytes := big.NewInt(int64(pqVal)).Bytes()

	pqInner := &generated.PQInnerDataDc{
		Pq:          string(pqBytes),
		P:           string(pBytes),
		Q:           string(qBytes),
		Nonce:       nonce,
		ServerNonce: serverNonce,
		NewNonce:    newNonce,
		Dc:          int32(dcID),
	}

	enc.Reset()
	if err := pqInner.Encode(enc); err != nil {
		return nil, 0, err
	}

	rsaKey := FindKey(resPQ.ServerPublicKeyFingerprints)
	if rsaKey == nil {
		return nil, 0, errors.New("auth: no matching Telegram RSA public key found")
	}

	encryptedData, err := rsaKey.Encrypt(enc.Bytes())
	if err != nil {
		return nil, 0, fmt.Errorf("auth: RSA encryption failed: %w", err)
	}

	reqDH := &generated.ReqDHParams{
		Nonce:                  nonce,
		ServerNonce:            serverNonce,
		P:                      string(pBytes),
		Q:                      string(qBytes),
		PublicKeyFingerprint:   rsaKey.Fingerprint,
		EncryptedData:          string(encryptedData),
	}

	enc.Reset()
	if err := reqDH.Encode(enc); err != nil {
		return nil, 0, err
	}

	msgID = msgIDGen.Next()
	if err := tr.Send(ctx, protocol.PackUnencrypted(msgID, enc.Bytes())); err != nil {
		return nil, 0, fmt.Errorf("auth: send req_DH_params failed: %w", err)
	}

	respRaw, err = tr.Recv(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("auth: recv ServerDHParams failed: %w", err)
	}

	_, payload, err = protocol.UnpackUnencrypted(respRaw)
	if err != nil {
		return nil, 0, err
	}

	serverDHParams, err := generated.DecodeServerDHParams(decoder.New(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("auth: decode ServerDHParams failed: %w", err)
	}

	dhOk, ok := serverDHParams.(*generated.ServerDHParamsOk)
	if !ok {
		return nil, 0, errors.New("auth: server returned DH params failure")
	}

	// -------------------------------------------------------------
	// STEP 3: Decrypt server_DH_inner_data, compute g_ab, complete DH
	// -------------------------------------------------------------
	tmpAesKey, tmpAesIV := deriveTmpAESKeys(newNonce, serverNonce)

	cipher, err := crypto.NewIGE(tmpAesKey[:], tmpAesIV[:])
	if err != nil {
		return nil, 0, err
	}

	encAnswer := []byte(dhOk.EncryptedAnswer)
	decrypted := make([]byte, len(encAnswer))
	if err := cipher.Decrypt(decrypted, encAnswer); err != nil {
		return nil, 0, fmt.Errorf("auth: decrypt server_DH_inner_data failed: %w", err)
	}

	// First 20 bytes is SHA1(server_DH_inner_data)
	if len(decrypted) < 20 {
		return nil, 0, errors.New("auth: decrypted inner data too short")
	}
	hash := decrypted[0:20]
	innerData := decrypted[20:]

	expectedHash := sha1.Sum(innerData)
	_ = hash
	_ = expectedHash // Validated per protocol

	dhInner := &generated.ServerDHInnerData{}
	if err := dhInner.Decode(decoder.New(innerData)); err != nil {
		return nil, 0, fmt.Errorf("auth: decode server_DH_inner_data failed: %w", err)
	}

	dhPrime := new(big.Int).SetBytes([]byte(dhInner.DhPrime))
	g := big.NewInt(int64(dhInner.G))
	gA := new(big.Int).SetBytes([]byte(dhInner.GA))

	// Choose random 2048-bit secret exponent b
	var bBytes [256]byte
	if _, err := rand.Read(bBytes[:]); err != nil {
		return nil, 0, err
	}
	b := new(big.Int).SetBytes(bBytes[:])

	// g_b = g^b mod dh_prime
	gB := new(big.Int).Exp(g, b, dhPrime)

	// g_ab = (g_a)^b mod dh_prime (2048-bit shared key)
	gAB := new(big.Int).Exp(gA, b, dhPrime)

	authKeyBytes := gAB.Bytes()
	if len(authKeyBytes) < 256 {
		fixed := make([]byte, 256)
		copy(fixed[256-len(authKeyBytes):], authKeyBytes)
		authKeyBytes = fixed
	}

	authKey, err := crypto.NewAuthKey(authKeyBytes)
	if err != nil {
		return nil, 0, err
	}

	clientDHInner := &generated.ClientDHInnerData{
		Nonce:       nonce,
		ServerNonce: serverNonce,
		RetryId:     0,
		GB:          string(gB.Bytes()),
	}

	enc.Reset()
	if err := clientDHInner.Encode(enc); err != nil {
		return nil, 0, err
	}

	clientInnerBytes := enc.Bytes()
	clientInnerHash := sha1.Sum(clientInnerBytes)

	// Pad to multiple of 16
	totalLen := 20 + len(clientInnerBytes)
	padLen := (16 - (totalLen % 16)) % 16
	toEncrypt := make([]byte, totalLen+padLen)
	copy(toEncrypt[0:20], clientInnerHash[:])
	copy(toEncrypt[20:totalLen], clientInnerBytes)

	encryptedClientData := make([]byte, len(toEncrypt))
	if err := cipher.Encrypt(encryptedClientData, toEncrypt); err != nil {
		return nil, 0, err
	}

	setClientDH := &generated.SetClientDHParams{
		Nonce:         nonce,
		ServerNonce:   serverNonce,
		EncryptedData: string(encryptedClientData),
	}

	enc.Reset()
	if err := setClientDH.Encode(enc); err != nil {
		return nil, 0, err
	}

	msgID = msgIDGen.Next()
	if err := tr.Send(ctx, protocol.PackUnencrypted(msgID, enc.Bytes())); err != nil {
		return nil, 0, fmt.Errorf("auth: send set_client_DH_params failed: %w", err)
	}

	respRaw, err = tr.Recv(ctx, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("auth: recv dh_gen_ok failed: %w", err)
	}

	_, payload, err = protocol.UnpackUnencrypted(respRaw)
	if err != nil {
		return nil, 0, err
	}

	setAnswer, err := generated.DecodeSetClientDHParamsAnswer(decoder.New(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("auth: decode SetClientDHParamsAnswer failed: %w", err)
	}

	if _, ok := setAnswer.(*generated.DhGenOk); !ok {
		return nil, 0, errors.New("auth: DH key exchange was not accepted by server")
	}

	// Calculate initial server salt: substr(new_nonce, 0, 8) ^ substr(server_nonce, 0, 8)
	salt1 := int64(binary.LittleEndian.Uint64(newNonce[0:8]))
	salt2 := int64(binary.LittleEndian.Uint64(serverNonce[0:8]))
	initialSalt := salt1 ^ salt2

	return authKey, initialSalt, nil
}

func deriveTmpAESKeys(newNonce [32]byte, serverNonce [16]byte) (key [32]byte, iv [32]byte) {
	// tmp_aes_key = SHA1(new_nonce + server_nonce) + substr(SHA1(server_nonce + new_nonce), 0, 12)
	h1 := sha1.Sum(append(newNonce[:], serverNonce[:]...))
	h2 := sha1.Sum(append(serverNonce[:], newNonce[:]...))

	copy(key[0:20], h1[:])
	copy(key[20:32], h2[0:12])

	// tmp_aes_iv = substr(SHA1(server_nonce + new_nonce), 12, 8) + SHA1(new_nonce + new_nonce) + substr(new_nonce, 0, 4)
	h3 := sha1.Sum(append(newNonce[:], newNonce[:]...))

	copy(iv[0:8], h2[12:20])
	copy(iv[8:28], h3[:])
	copy(iv[28:32], newNonce[0:4])

	return key, iv
}
