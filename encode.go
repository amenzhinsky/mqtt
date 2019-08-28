package mqtt

import (
	"errors"
	"io"
	"unicode/utf8"

	"github.com/amenzhinsky/mqtt/packet"
)

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type Encoder struct {
	w io.Writer
	b enc
}

func (e *Encoder) Encode(pk packet.OutgoingPacket) error {
	e.b.reset()
	e.b = append(e.b, byte(pk.GetFlags()))
	var err error
	if err = pk.Encode(&e.b); err != nil {
		return err
	}
	_, err = e.w.Write(e.b)
	return err
}

type enc []byte

func (e *enc) reset() {
	if e == nil {
		*e = make([]byte, 0, 4096)
	} else {
		*e = (*e)[:0]
	}
}

func (e *enc) Len(n int) error {
	const maxLen = 1024*1024*256 - 1 // 256MB
	if n > maxLen {
		return errors.New("length is too big")
	}
	for i := 1; ; i++ {
		c := uint8(n % 128)
		n /= 128
		if n > 0 {
			c |= 128
		}
		*e = append(*e, c)
		if n == 0 {
			return nil
		}
	}
}

func (e *enc) Bits(c uint8) error {
	*e = append(*e, c)
	return nil
}

func (e *enc) Integer(n uint16) error {
	*e = append(*e, uint8(n>>8), uint8(n))
	return nil
}

func (e *enc) Payload(b []byte) error {
	*e = append(*e, b...)
	return nil
}

const maxUint16 = 1<<16 - 1

func (e *enc) Bytes(b []byte) error {
	if len(b) > maxUint16 {
		return errors.New("too long bytes array")
	}
	if err := e.Integer(uint16(len(b))); err != nil {
		return err
	}
	*e = append(*e, b...)
	return nil
}

func (e *enc) String(s string) error {
	if !utf8.ValidString(s) {
		return errors.New("invalid utf-8 string")
	}
	if len(s) > maxUint16 {
		return errors.New("too long string")
	}
	if err := e.Integer(uint16(len(s))); err != nil {
		return err
	}
	*e = append(*e, s...)
	return nil
}
