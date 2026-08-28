package encoder

// Constants for core TL combinator IDs.
const (
	CRCVector    uint32 = 0x1cb5c415
	CRCBoolTrue  uint32 = 0x997275b5
	CRCBoolFalse uint32 = 0xbc799737
)

// Encoder serializes Telegram Type Language (TL) data into binary format.
// It manages a contiguous byte buffer and provides methods for serializing
// TL primitive and composite types without reflection.
type Encoder struct {
	buf []byte
}

// New creates an Encoder with an initial capacity of 256 bytes.
func New() *Encoder {
	return &Encoder{
		buf: make([]byte, 0, 256),
	}
}

// NewWithBuffer creates an Encoder that reuses the provided byte slice.
// The slice is reset to zero length while preserving its underlying capacity.
func NewWithBuffer(buf []byte) *Encoder {
	return &Encoder{
		buf: buf[:0],
	}
}

// Reset resets the encoder's buffer to zero length without deallocating
// the underlying storage, enabling zero-allocation reuse.
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}

// Bytes returns the serialized bytes. The returned slice is valid until
// the next write or Reset call.
func (e *Encoder) Bytes() []byte {
	return e.buf
}

// Len returns the current number of encoded bytes.
func (e *Encoder) Len() int {
	return len(e.buf)
}

// Grow ensures that there is space for at least n more bytes.
func (e *Encoder) Grow(n int) {
	if cap(e.buf)-len(e.buf) < n {
		newCap := 2*cap(e.buf) + n
		newBuf := make([]byte, len(e.buf), newCap)
		copy(newBuf, e.buf)
		e.buf = newBuf
	}
}

// PutByte appends a single raw byte.
func (e *Encoder) PutByte(b byte) {
	e.buf = append(e.buf, b)
}

// PutRaw appends raw bytes directly without length prefix or padding.
func (e *Encoder) PutRaw(b []byte) {
	e.buf = append(e.buf, b...)
}
