package encoder

import (
	"encoding/binary"
	"math"
)

var zeroes [4]byte

// PutInt writes a 32-bit signed integer in little-endian format.
func (e *Encoder) PutInt(v int32) {
	e.PutUint(uint32(v))
}

// PutUint writes a 32-bit unsigned integer in little-endian format.
func (e *Encoder) PutUint(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	e.buf = append(e.buf, b[:]...)
}

// PutLong writes a 64-bit signed integer in little-endian format.
func (e *Encoder) PutLong(v int64) {
	e.PutUint64(uint64(v))
}

// PutUint64 writes a 64-bit unsigned integer in little-endian format.
func (e *Encoder) PutUint64(v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	e.buf = append(e.buf, b[:]...)
}

// PutDouble writes a 64-bit floating-point number in little-endian IEEE 754 format.
func (e *Encoder) PutDouble(v float64) {
	e.PutUint64(math.Float64bits(v))
}

// PutBool writes a TL boolean constructor ID (boolTrue or boolFalse).
func (e *Encoder) PutBool(v bool) {
	if v {
		e.PutUint(CRCBoolTrue)
	} else {
		e.PutUint(CRCBoolFalse)
	}
}

// PutInt128 writes a 128-bit integer (16 raw bytes).
func (e *Encoder) PutInt128(v [16]byte) {
	e.buf = append(e.buf, v[:]...)
}

// PutInt256 writes a 256-bit integer (32 raw bytes).
func (e *Encoder) PutInt256(v [32]byte) {
	e.buf = append(e.buf, v[:]...)
}

// PutBytes writes a byte string according to TL serialization rules:
// - If len <= 253: 1 length byte, payload, and null padding to a 4-byte boundary.
// - If len >= 254: 0xFE byte, 3-byte little-endian length, payload, and null padding to a 4-byte boundary.
func (e *Encoder) PutBytes(v []byte) {
	n := len(v)
	if n <= 253 {
		e.buf = append(e.buf, byte(n))
		e.buf = append(e.buf, v...)
		pad := (4 - ((1 + n) % 4)) % 4
		if pad > 0 {
			e.buf = append(e.buf, zeroes[:pad]...)
		}
	} else {
		var hdr [4]byte
		hdr[0] = 0xfe
		hdr[1] = byte(n)
		hdr[2] = byte(n >> 8)
		hdr[3] = byte(n >> 16)
		e.buf = append(e.buf, hdr[:]...)
		e.buf = append(e.buf, v...)
		pad := (4 - (n % 4)) % 4
		if pad > 0 {
			e.buf = append(e.buf, zeroes[:pad]...)
		}
	}
}

// PutString writes a string according to TL byte string serialization rules.
func (e *Encoder) PutString(s string) {
	n := len(s)
	if n <= 253 {
		e.buf = append(e.buf, byte(n))
		e.buf = append(e.buf, s...)
		pad := (4 - ((1 + n) % 4)) % 4
		if pad > 0 {
			e.buf = append(e.buf, zeroes[:pad]...)
		}
	} else {
		var hdr [4]byte
		hdr[0] = 0xfe
		hdr[1] = byte(n)
		hdr[2] = byte(n >> 8)
		hdr[3] = byte(n >> 16)
		e.buf = append(e.buf, hdr[:]...)
		e.buf = append(e.buf, s...)
		pad := (4 - (n % 4)) % 4
		if pad > 0 {
			e.buf = append(e.buf, zeroes[:pad]...)
		}
	}
}

// PutVectorHeader writes the standard TL vector constructor ID (0x1cb5c415)
// followed by the 32-bit element count.
func (e *Encoder) PutVectorHeader(count int) {
	e.PutUint(CRCVector)
	e.PutInt(int32(count))
}
