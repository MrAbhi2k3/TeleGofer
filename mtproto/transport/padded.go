package transport

import (
	"crypto/rand"
	"encoding/binary"
	"io"
)

// Padded implements Telegram's Padded Intermediate TCP transport.
// Each packet includes 0-15 random padding bytes to deter traffic analysis.
// Initial protocol tag: 0xdddddddd.
type Padded struct{}

// NewPadded creates a Padded Intermediate transport codec.
func NewPadded() *Padded {
	return &Padded{}
}

// Handshake sends the 0xdddddddd protocol tag.
func (p *Padded) Handshake(w io.Writer) error {
	tag := []byte{0xdd, 0xdd, 0xdd, 0xdd}
	_, err := w.Write(tag)
	return err
}

// WritePacket appends random padding (0-15 bytes) and transmits the packet
// with a 4-byte length prefix.
func (p *Padded) WritePacket(w io.Writer, payload []byte) error {
	var padCountBuf [1]byte
	if _, err := rand.Read(padCountBuf[:]); err != nil {
		return err
	}
	padLen := int(padCountBuf[0] & 0x0f) // 0 to 15 bytes

	var pad [16]byte
	if padLen > 0 {
		if _, err := rand.Read(pad[:padLen]); err != nil {
			return err
		}
	}

	totalLen := len(payload) + padLen
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(totalLen))

	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	if padLen > 0 {
		if _, err := w.Write(pad[:padLen]); err != nil {
			return err
		}
	}

	return nil
}

// ReadPacket reads a single padded intermediate frame from the reader.
func (p *Padded) ReadPacket(r io.Reader, buf []byte) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}

	length := int(binary.LittleEndian.Uint32(lenBuf[:]))
	if length <= 0 || length > 16*1024*1024 {
		return nil, ErrInvalidLength
	}

	out := buf
	if cap(out) >= length {
		out = out[:length]
	} else {
		out = make([]byte, length)
	}

	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}

	return out, nil
}
