package transport

import (
	"io"
)

// Abridged implements Telegram's lightweight Abridged TCP transport protocol.
// Initial protocol tag: 0xef.
type Abridged struct{}

// NewAbridged creates an Abridged transport codec.
func NewAbridged() *Abridged {
	return &Abridged{}
}

// Handshake sends the 0xef protocol tag.
func (a *Abridged) Handshake(w io.Writer) error {
	_, err := w.Write([]byte{0xef})
	return err
}

// WritePacket frames an MTProto payload into the abridged envelope.
func (a *Abridged) WritePacket(w io.Writer, payload []byte) error {
	n := len(payload)
	if n%4 != 0 {
		return ErrInvalidLength
	}

	len4 := n / 4
	if len4 < 127 {
		hdr := []byte{byte(len4)}
		if _, err := w.Write(hdr); err != nil {
			return err
		}
	} else {
		hdr := []byte{
			0x7f,
			byte(len4),
			byte(len4 >> 8),
			byte(len4 >> 16),
		}
		if _, err := w.Write(hdr); err != nil {
			return err
		}
	}

	_, err := w.Write(payload)
	return err
}

// ReadPacket reads a single abridged frame from the reader.
func (a *Abridged) ReadPacket(r io.Reader, buf []byte) ([]byte, error) {
	var first [1]byte
	if _, err := io.ReadFull(r, first[:]); err != nil {
		return nil, err
	}

	var length int
	if first[0] < 0x7f {
		length = int(first[0]) * 4
	} else {
		var lenBytes [3]byte
		if _, err := io.ReadFull(r, lenBytes[:]); err != nil {
			return nil, err
		}
		len4 := int(lenBytes[0]) | (int(lenBytes[1]) << 8) | (int(lenBytes[2]) << 16)
		length = len4 * 4
	}

	if length <= 0 || length > 16*1024*1024 { // Sanity bound: 16 MB max MTProto packet
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
