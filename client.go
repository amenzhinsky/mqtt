package mqtt

import (
	"io"
	"log"
)

type Option func(c *Client)

func WithLogger(l *log.Logger) Option {
	return func(c *Client) {
		c.l = l
	}
}

func NewClient(rw io.ReadWriteCloser, opts ...Option) *Client {
	c := &Client{
		e: NewEncoder(rw),
		d: NewDecoder(rw),
		c: rw,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Client struct {
	e *Encoder
	d *Decoder
	c io.Closer
	l *log.Logger
}

func (c *Client) Send(pk OutgoingPacket) error {
	if err := c.e.Encode(pk); err != nil {
		return err
	}
	if c.l != nil {
		c.l.Print(">", pk.String())
	}
	return nil
}

func (c *Client) Recv() (IncomingPacket, error) {
	pk, err := c.d.Decode()
	if err != nil {
		return nil, err
	}
	if c.l != nil {
		c.l.Print("<", pk.String())
	}
	return pk, nil
}

func (c *Client) Close() error {
	return c.c.Close()
}
