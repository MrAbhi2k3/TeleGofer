package transport

import (
	"encoding/binary"
	"hash/crc32"
	"io"
	"sync"
)

// Full implements Telegram's Full TCP transport protocol with sequence numbers
// and end-to-end CRC32 checksums on every packet.
type Full struct {
	sendSeq uint32
	recvSeq uint32
	sendMu  sync.Mutex
	recvMu  sync.Mutex
}

// NewFull creates a Full transport codec.
func NewFull() *Full {
	return &Full{}
}

// Handshake is a no-op for Full transport since packet headers serve as self-identification.
func (f *Full) Handshake(w io.Writer) error {
	return nil
}

// WritePacket frames an MTProto payload with packet length, sequence number, and CRC32 checksum.
func (f *Full) WritePacket(w io.Writer, payload []byte) error {
	f.sendMu.Lock()
	seq := f.sendSeq
	f.sendSeq++
	f.sendMu.Unlock()

	totalLen := len(payload) + 12 // 4 (length) + 4 (seq) + payload + 4 (crc)

	buf := make([]byte, totalLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(totalLen))
	binary.LittleEndian.PutUint32(buf[4:8], seq)
	copy(buf[8:8+len(payload)], payload)

	checksum := crc32.ChecksumIEEE(buf[:8+len(payload)])
	binary.LittleEndian.PutUint32(buf[8+len(payload):], checksum)

	_, err := w.Write(buf)
	return err
}

// ReadPacket reads a single Full frame from the reader and verifies its CRC32 checksum.
func (f *Full) ReadPacket(r io.Reader, buf []byte) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}

	totalLen := int(binary.LittleEndian.Uint32(lenBuf[:]))
	if totalLen < 12 || totalLen > 16*1024*1024 {
		return nil, ErrInvalidLength
	}

	payloadLen := totalLen - 12

	// Read remaining bytes: 4 (seq) + payloadLen + 4 (crc)
	raw := make([]byte, totalLen)
	copy(raw[0:4], lenBuf[:])

	if _, err := io.ReadFull(r, raw[4:]); err != nil {
		return nil, err
	}

	// Verify CRC32
	expectedCRC := binary.LittleEndian.Uint32(raw[totalLen-4:])
	calculatedCRC := crc32.ChecksumIEEE(raw[:totalLen-4])
	if expectedCRC != calculatedCRC {
		return nil, ErrCRCMismatch
	}

	f.recvMu.Lock()
	f.recvSeq++
	f.recvMu.Unlock()

	out := buf
	if cap(out) >= payloadLen {
		out = out[:payloadLen]
	} else {
		out = make([]byte, payloadLen)
	}

	copy(out, raw[8:8+payloadLen])
	return out, nil
}
