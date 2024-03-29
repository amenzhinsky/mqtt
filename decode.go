package mqtt

import (
	"errors"
	"fmt"
	"io"

	"github.com/amenzhinsky/mqtt/packet"
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{dec: &dec{buf: &buffer{r: r}}}
}

type Decoder struct {
	dec *dec
}

func (d *Decoder) Decode() (packet.IncomingPacket, error) {
	// read fixed reader to determine packet type
	h, err := d.dec.buf.Byte()
	if err != nil {
		return nil, err
	}
	if err = d.dec.readLenAndGrow(); err != nil {
		return nil, err
	}

	pk := packet.NewIncomingPacket(h)
	if pk == nil {
		return nil, fmt.Errorf("unknown packet type 0x%x", h>>4)
	}
	if err = pk.Decode(d.dec); err != nil {
		return nil, err
	}
	if d.dec.len != 0 {
		return nil, fmt.Errorf("unread bytes remaining: %d", d.dec.len)
	}
	return pk, nil
}

type dec struct {
	buf *buffer
	len int
}

func (d *dec) Bits() (byte, error) {
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

func (d *dec) Integer() (uint16, error) {
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

func (d *dec) Payload() ([]byte, error) {
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

func (d *dec) Bytes() ([]byte, error) {
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
	// dec reuses its buffer that may cause implicit changes in packets
	v := make([]byte, n)
	d.len -= copy(v, b)
	return v, nil
}

func (d *dec) String() (string, error) {
	b, err := d.Bytes()
	if err != nil {
		return "", nil
	}
	return string(b), nil
}

func (d *dec) checkAvailableBytes(n int) error {
	if d.len < n {
		return fmt.Errorf("malformed packet, len=%d want=%d", d.len, n)
	}
	return nil
}

// reads the remaining length and advances the underlying buffer to fit whole packet
func (d *dec) readLenAndGrow() error {
	size, err := d.readLen()
	if err != nil {
		return err
	}
	d.len = size
	return d.buf.Grow(size)
}

func (d *dec) readLen() (int, error) {
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
