package transport

import (
	"encoding/binary"
	"io"
)

// Intermediate implements Telegram's 4-byte-aligned Intermediate TCP transport.
// Initial protocol tag: 0xeeeeeeee.
type Intermediate struct{}

// NewIntermediate creates an Intermediate transport codec.
func NewIntermediate() *Intermediate {
	return &Intermediate{}
}

// Handshake sends the 0xeeeeeeee protocol tag.
func (t *Intermediate) Handshake(w io.Writer) error {
	tag := []byte{0xee, 0xee, 0xee, 0xee}
	_, err := w.Write(tag)
	return err
}

// WritePacket frames an MTProto payload with a 4-byte little-endian length prefix.
func (t *Intermediate) WritePacket(w io.Writer, payload []byte) error {
	var lenBuf [4]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadPacket reads a single intermediate frame from the reader.
func (t *Intermediate) ReadPacket(r io.Reader, buf []byte) ([]byte, error) {
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
