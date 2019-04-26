package mqtt

import (
	"errors"
	"io"
	"sync"
	"unicode/utf8"
)

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

type Encoder struct {
	m sync.Mutex
	w io.Writer
	b encoder
}

func (e *Encoder) Encode(pk OutgoingPacket) error {
	e.m.Lock()
	defer e.m.Unlock()

	// reserve enough space for flags and the remaining length
	e.b.reset()
	e.b = append(e.b, byte(pk.flags()), 0, 0, 0, 0)

	var err error
	if err = pk.encode(&e.b); err != nil {
		return err
	}

	// body is empty, DISCONNECT, PINGREQ, etc.
	if len(e.b) == 5 {
		_, err = e.w.Write(e.b[:2])
		return err
	}

	if err = encodeLen(e.b, len(e.b)-5); err != nil {
		return err
	}

	// shift fixed header's bytes to the right
	// to write the whole packet in one write call
	var offset int
	for i := 1; i < 5; i++ {
		if e.b[i] != 0 {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			e.b[j+5-i] = e.b[j]
		}
		offset = 5 - i
		break
	}
	_, err = e.w.Write(e.b[offset:])
	return err
}

func encodeLen(fh []byte, size int) error {
	const maxLen = 1024*1024*256 - 1 // 256MB
	if size > maxLen {
		return errors.New("length is too big")
	}
	for i := 1; ; i++ {
		c := uint8(size % 128)
		size /= 128
		if size > 0 {
			c |= 128
		}
		fh[i] = c
		if size == 0 {
			return nil
		}
	}
}

type encoder []byte

func (e *encoder) reset() {
	if e == nil {
		*e = make([]byte, 0, 4096)
	} else {
		*e = (*e)[:0]
	}
}

func (e *encoder) Bits(c uint8) error {
	*e = append(*e, c)
	return nil
}

func (e *encoder) Integer(n uint16) error {
	*e = append(*e, uint8(n>>8), uint8(n))
	return nil
}

func (e *encoder) Payload(b []byte) error {
	*e = append(*e, b...)
	return nil
}

const maxUint16 = 1<<16 - 1

func (e *encoder) Bytes(b []byte) error {
	if len(b) > maxUint16 {
		return errors.New("too long bytes array")
	}
	if err := e.Integer(uint16(len(b))); err != nil {
		return err
	}
	*e = append(*e, b...)
	return nil
}

func (e *encoder) String(s string) error {
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
