package decoder

import (
	"encoding/binary"
	"math"
)

func (d *Decoder) Uint() (uint32, error) {
	if d.Remaining() < 4 {
		return 0, ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint32(d.buf[d.pos : d.pos+4])
	d.pos += 4
	return v, nil
}

func (d *Decoder) Int() (int32, error) {
	v, err := d.Uint()
	return int32(v), err
}

func (d *Decoder) Long() (int64, error) {
	v, err := d.Uint64()
	return int64(v), err
}

func (d *Decoder) Uint64() (uint64, error) {
	if d.Remaining() < 8 {
		return 0, ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint64(d.buf[d.pos : d.pos+8])
	d.pos += 8
	return v, nil
}

func (d *Decoder) Double() (float64, error) {
	v, err := d.Uint64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

// Bool reads a TL boolean constructor (boolTrue or boolFalse).
func (d *Decoder) Bool() (bool, error) {
	crc, err := d.Uint()
	if err != nil {
		return false, err
	}
	switch crc {
	case CRCBoolTrue:
		return true, nil
	case CRCBoolFalse:
		return false, nil
	default:
		return false, ErrInvalidBool
	}
}

// Int128 reads a 128-bit integer (16 raw bytes).
func (d *Decoder) Int128() ([16]byte, error) {
	var out [16]byte
	if d.Remaining() < 16 {
		return out, ErrUnexpectedEOF
	}
	copy(out[:], d.buf[d.pos:d.pos+16])
	d.pos += 16
	return out, nil
}

// Int256 reads a 256-bit integer (32 raw bytes).
func (d *Decoder) Int256() ([32]byte, error) {
	var out [32]byte
	if d.Remaining() < 32 {
		return out, ErrUnexpectedEOF
	}
	copy(out[:], d.buf[d.pos:d.pos+32])
	d.pos += 32
	return out, nil
}

func (d *Decoder) Raw(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeLength
	}
	if d.Remaining() < n {
		return nil, ErrUnexpectedEOF
	}
	out := make([]byte, n)
	copy(out, d.buf[d.pos:d.pos+n])
	d.pos += n
	return out, nil
}

func (d *Decoder) RawBorrow(n int) ([]byte, error) {
	if n < 0 {
		return nil, ErrNegativeLength
	}
	if d.Remaining() < n {
		return nil, ErrUnexpectedEOF
	}
	out := d.buf[d.pos : d.pos+n]
	d.pos += n
	return out, nil
}

func (d *Decoder) Bytes() ([]byte, error) {
	data, err := d.readBytesView()
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func (d *Decoder) BytesBorrow() ([]byte, error) {
	return d.readBytesView()
}

// String reads a TL byte string as a Go string.
func (d *Decoder) String() (string, error) {
	data, err := d.readBytesView()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// readBytesView parses the TL byte string framing and advances pos past the data and padding.
func (d *Decoder) readBytesView() ([]byte, error) {
	if d.Remaining() < 1 {
		return nil, ErrUnexpectedEOF
	}
	first := d.buf[d.pos]

	var length int
	var headerLen int
	var pad int

	if first <= 253 {
		length = int(first)
		headerLen = 1
		pad = (4 - ((1 + length) % 4)) % 4
	} else {
		// 0xfe followed by 3 bytes little-endian length
		if d.Remaining() < 4 {
			return nil, ErrUnexpectedEOF
		}
		length = int(d.buf[d.pos+1]) | (int(d.buf[d.pos+2]) << 8) | (int(d.buf[d.pos+3]) << 16)
		headerLen = 4
		pad = (4 - (length % 4)) % 4
	}

	total := headerLen + length + pad
	if d.Remaining() < total {
		return nil, ErrUnexpectedEOF
	}

	// Validate padding bytes are all zeroes per specification
	padStart := d.pos + headerLen + length
	for i := 0; i < pad; i++ {
		if d.buf[padStart+i] != 0 {
			return nil, ErrInvalidPadding
		}
	}

	dataStart := d.pos + headerLen
	out := d.buf[dataStart : dataStart+length]
	d.pos += total
	return out, nil
}

func (d *Decoder) VectorHeader() (int, error) {
	crc, err := d.Uint()
	if err != nil {
		return 0, err
	}
	if crc != CRCVector {
		return 0, ErrInvalidVector
	}
	count, err := d.Int()
	if err != nil {
		return 0, err
	}
	if count < 0 {
		return 0, ErrNegativeLength
	}
	// Sanity check: each TL element takes at least 4 bytes on wire
	if int(count) > d.Remaining()/4 {
		return 0, ErrLengthTooLarge
	}
	return int(count), nil
}

func (d *Decoder) PeekUint() (uint32, error) {
	if d.Remaining() < 4 {
		return 0, ErrUnexpectedEOF
	}
	return binary.LittleEndian.Uint32(d.buf[d.pos : d.pos+4]), nil
}
