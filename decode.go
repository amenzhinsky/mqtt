package mqtt

import (
	"errors"
	"fmt"
	"io"
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{dec: &decoder{buf: &buffer{r: r}}}
}

type Decoder struct {
	dec *decoder
}

func (d *Decoder) Decode() (IncomingPacket, error) {
	// read fixed reader to determine packet type
	h, err := d.dec.buf.Byte()
	if err != nil {
		return nil, err
	}
	if err = d.dec.readLenAndGrow(); err != nil {
		return nil, err
	}

	var pk IncomingPacket
	switch h & 0xf0 {
	case pkConnack:
		pk = &Connack{Flags: Flags(h)}
	case pkPublish:
		pk = &Publish{Flags: Flags(h)}
	case pkPuback:
		pk = &Puback{Flags: Flags(h)}
	case pkPubrec:
		pk = &Pubrec{Flags: Flags(h)}
	case pkPubcomp:
		pk = &Pubcomp{Flags: Flags(h)}
	case pkSuback:
		pk = &Suback{Flags: Flags(h)}
	case pkUnsuback:
		pk = &Unsuback{Flags: Flags(h)}
	case pkPingresp:
		pk = &Pingresp{Flags: Flags(h)}
	default:
		return nil, fmt.Errorf("unknown packet type %d", uint8(h>>4))
	}
	if err = pk.decode(d.dec); err != nil {
		return nil, err
	}
	if d.dec.len != 0 {
		return nil, fmt.Errorf("unread bytes remaining: %d", d.dec.len)
	}
	return pk, nil
}

type decoder struct {
	buf *buffer
	len int
}

func (d *decoder) Bits() (byte, error) {
	if err := d.checkAvailableBytes(1); err != nil {
		return 0, err
	}
	c, err := d.buf.Byte()
	if err != nil {
		return 0, err
	}
	d.len -= 1
	return c, nil
}

func (d *decoder) Integer() (uint16, error) {
	if err := d.checkAvailableBytes(2); err != nil {
		return 0, err
	}
	b1, err := d.buf.Byte()
	if err != nil {
		return 0, err
	}
	b2, err := d.buf.Byte()
	if err != nil {
		return 0, err
	}
	d.len -= 2
	return uint16(b2) | uint16(b1)<<8, nil
}

func (d *decoder) Payload() ([]byte, error) {
	if err := d.checkAvailableBytes(d.len); err != nil {
		return nil, err
	}
	b, err := d.buf.Bytes(d.len)
	if err != nil {
		return nil, err
	}
	d.len = 0
	return b, nil
}

func (d *decoder) Bytes() ([]byte, error) {
	n, err := d.Integer()
	if err != nil {
		return nil, err
	}
	if err := d.checkAvailableBytes(int(n)); err != nil {
		return nil, err
	}
	b, err := d.buf.Bytes(int(n))
	if err != nil {
		return nil, err
	}
	// decoder reuses its buffer that may cause implicit changes in packets
	v := make([]byte, n)
	d.len -= copy(v, b)
	return v, nil
}

func (d *decoder) String() (string, error) {
	b, err := d.Bytes()
	if err != nil {
		return "", nil
	}
	return string(b), nil
}

func (d *decoder) checkAvailableBytes(n int) error {
	if d.len < n {
		return errors.New("malformed packet")
	}
	return nil
}

// reads the remaining length and advances the underlying buffer to fit whole packet
func (d *decoder) readLenAndGrow() error {
	size, err := d.readLen()
	if err != nil {
		return err
	}
	if size == 0 {
		return errors.New("malformed packet")
	}
	d.len = size
	return d.buf.Grow(size)
}

func (d *decoder) readLen() (int, error) {
	const maxMul = 128 * 128 * 128

	m := 1
	v := 0
	for {
		b, err := d.buf.Byte()
		if err != nil {
			return 0, err
		}
		v += int(b&127) * m
		m *= 128
		if m > maxMul {
			return 0, errors.New("malformed length")
		}
		if b&128 == 0 {
			return v, nil
		}
	}
}

type buffer struct {
	r   io.Reader
	buf []byte
	off int
	end int
}

func (b *buffer) Byte() (byte, error) {
	if err := b.Grow(1); err != nil {
		return 0, err
	}
	c := b.buf[b.off]
	b.off++
	return c, nil
}

func (b *buffer) Bytes(size int) ([]byte, error) {
	if err := b.Grow(size); err != nil {
		return nil, err
	}
	v := b.buf[b.off : b.off+size]
	b.off += size
	return v, nil
}

func (b *buffer) Grow(size int) error {
	if size < 0 {
		panic("size is negative")
	}
	if size <= b.end-b.off {
		return nil
	}

	// grow buffer or shift bytes to the left to make space for next read
	if size > len(b.buf) {
		capacity := size
		if capacity < 4096 {
			capacity = 4096
		}
		n := make([]byte, capacity)
		b.end = copy(n, b.buf[b.off:])
		b.off = 0
		b.buf = n
	} else if size > len(b.buf)-b.off {
		b.end = copy(b.buf, b.buf[b.off:])
		b.off = 0
	}

	for {
		n, err := b.r.Read(b.buf[b.end:])
		if err != nil {
			return err
		}
		b.end += n
		if size <= b.end-b.off {
			return nil
		}
	}
}
