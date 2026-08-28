package decoder

import "errors"

// Sentinel errors returned by Decoder operations.
var (
	ErrUnexpectedEOF  = errors.New("tl/decoder: unexpected end of buffer")
	ErrInvalidBool    = errors.New("tl/decoder: invalid boolean constructor")
	ErrInvalidVector  = errors.New("tl/decoder: invalid vector constructor")
	ErrNegativeLength = errors.New("tl/decoder: negative length encountered")
	ErrLengthTooLarge = errors.New("tl/decoder: length exceeds available buffer")
	ErrInvalidPadding = errors.New("tl/decoder: non-zero padding byte")
)

// Constants for core TL combinator IDs.
const (
	CRCVector    uint32 = 0x1cb5c415
	CRCBoolTrue  uint32 = 0x997275b5
	CRCBoolFalse uint32 = 0xbc799737
)

type Decoder struct {
	buf []byte
	pos int
}

func New(data []byte) *Decoder {
	return &Decoder{
		buf: data,
		pos: 0,
	}
}

func (d *Decoder) Reset(data []byte) {
	d.buf = data
	d.pos = 0
}

func (d *Decoder) Remaining() int {
	return len(d.buf) - d.pos
}

// Consumed returns the number of bytes read so far.
func (d *Decoder) Consumed() int {
	return d.pos
}

func (d *Decoder) BytesRemaining() []byte {
	return d.buf[d.pos:]
}

func (d *Decoder) Skip(n int) error {
	if n < 0 || d.Remaining() < n {
		return ErrUnexpectedEOF
	}
	d.pos += n
	return nil
}
