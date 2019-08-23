package mqtt

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type Option func(c *Client)

func WithWarnLogger(logger Logger) Option {
	return func(c *Client) {
		c.warn = logger
	}
}

func WithDebugLogger(logger Logger) Option {
	return func(c *Client) {
		c.debug = logger
	}
}

func WithHandler(handler HandlerFunc) Option {
	return func(c *Client) {
		c.handler = handler
	}
}

func NewClient(rw io.ReadWriteCloser, opts ...Option) *Client {
	c := &Client{
		e: NewEncoder(rw),
		d: NewDecoder(rw),
		c: rw,

		outc: make(chan *out),
		done: make(chan struct{}),

		connackc:  make(chan *Connack),
		pingrespc: make(chan *Pingresp),
		subackc:   make(chan *Suback),
		unsubackc: make(chan *Unsuback),
		pubackc:   make(chan *Puback),
		pubrecc:   make(chan *Pubrec),
		pubcompc:  make(chan *Pubcomp),
	}
	for _, opt := range opts {
		opt(c)
	}
	go c.in()
	go c.out()
	return c
}

type HandlerFunc func(publish *Publish)

type Client struct {
	e *Encoder
	d *Decoder
	c io.Closer

	warn  Logger
	debug Logger

	outc    chan *out
	done    chan struct{}
	err     error
	handler HandlerFunc

	connackc  chan *Connack
	pingrespc chan *Pingresp
	subackc   chan *Suback
	unsubackc chan *Unsuback
	pubackc   chan *Puback
	pubrecc   chan *Pubrec
	pubcompc  chan *Pubcomp
}

type out struct {
	pk   OutgoingPacket
	done chan struct{}
}

type Logger interface {
	Print(v ...interface{})
}

func (c *Client) Connect(ctx context.Context, connect *Connect) (*Connack, error) {
	if err := c.send(ctx, connect); err != nil {
		return nil, err
	}
	select {
	case connack := <-c.connackc:
		if connack.ReturnCode != ConnectionAccepted {
			return nil, fmt.Errorf("connection failed: %s", connack.ReturnCode.String())
		}
		return connack, nil
	case <-c.done:
		return nil, c.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.send(ctx, NewPingreqPacket()); err != nil {
		return err
	}
	select {
	case <-c.pingrespc:
		return nil
	case <-c.done:
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.send(ctx, NewDisconnectPacket())
}

var errInvalidPacketID = errors.New("invalid packet id")

func (c *Client) Publish(ctx context.Context, publish *Publish) error {
	if err := c.send(ctx, publish); err != nil {
		return err
	}
	switch {
	case enabled(uint8(publish.Flags), PublishQoS1):
		select {
		case puback := <-c.pubackc:
			if puback.PacketID != publish.PacketID {
				return errInvalidPacketID
			}
			return nil
		case <-c.done:
			return c.err
		}
	case enabled(uint8(publish.Flags), PublishQoS2):
		select {
		case pubrec := <-c.pubrecc:
			if pubrec.PacketID != publish.PacketID {
				return errInvalidPacketID
			}
			if err := c.send(ctx, NewPubrelPacket(publish.PacketID)); err != nil {
				return err
			}
			select {
			case pubcomp := <-c.pubcompc:
				if pubcomp.PacketID != publish.PacketID {
					return errInvalidPacketID
				}
				return nil
			case <-c.done:
				return c.err
			}
		case <-c.done:
			return c.err
		}
	default:
		return nil
	}
}

func (c *Client) Subscribe(ctx context.Context, subscribe *Subscribe) (*Suback, error) {
	if err := c.send(ctx, subscribe); err != nil {
		return nil, err
	}
	select {
	case suback := <-c.subackc:
		if suback.PacketID != subscribe.PacketID {
			return nil, errInvalidPacketID
		}
		return suback, nil
	case <-c.done:
		return nil, c.err
	}
}

func (c *Client) Unsubscribe(ctx context.Context, unsubscribe *Unsubscribe) error {
	if err := c.send(ctx, unsubscribe); err != nil {
		return err
	}
	select {
	case unsuback := <-c.unsubackc:
		if unsuback.PacketID != unsubscribe.PacketID {
			return errInvalidPacketID
		}
		return nil
	case <-c.done:
		return c.err
	}
}

func (c *Client) in() {
	for {
		pk, err := c.d.Decode()
		if err != nil {
			c.close(err)
			return
		}
		c.debugf("< %s", pk)
		switch v := pk.(type) {
		case *Connack:
			select {
			case c.connackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *Pingresp:
			select {
			case c.pingrespc <- v:
			case <-c.done:
				c.warnf("unexpected: %s", v)
			}
		case *Puback:
			select {
			case c.pubackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *Pubrec:
			select {
			case c.pubrecc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *Pubcomp:
			select {
			case c.pubcompc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *Suback:
			select {
			case c.subackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *Unsuback:
			select {
			case c.unsubackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *Publish:
			if c.handler != nil {
				c.handler(v)
			} else {
				c.warnf("unhandled: %s", v)
			}
		default:
			panic(fmt.Sprintf("unknown incomming packet: %#v", v))
		}
	}
}

func (c *Client) out() {
	for pk := range c.outc {
		if err := c.e.Encode(pk.pk); err != nil {
			c.close(err)
			return
		}
		c.debugf("> %s", pk.pk)
		close(pk.done)
	}
}

func (c *Client) warnf(format string, v ...interface{}) {
	if c.warn != nil {
		c.warn.Print(fmt.Sprintf(format, v...))
	}
}

func (c *Client) debugf(format string, v ...interface{}) {
	if c.debug != nil {
		c.debug.Print(fmt.Sprintf(format, v...))
	}
}

func (c *Client) send(ctx context.Context, pk OutgoingPacket) error {
	o := &out{pk: pk, done: make(chan struct{})}
	select {
	case c.outc <- o:
	case <-c.done:
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-o.done:
		return nil
	case <-c.done:
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) close(err error) {
	select {
	case <-c.done:
	default:
		c.err = err
		close(c.done)
	}
}

func (c *Client) Close() error {
	return c.c.Close()
}
