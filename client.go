package mqtt

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/amenzhinsky/mqtt/packet"
)

// Option is a client configuration option.
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

func newEncoderDecoder(rw io.ReadWriteCloser) *encoderDecoder {
	return &encoderDecoder{NewEncoder(rw), NewDecoder(rw)}
}

type encoderDecoder struct {
	*Encoder
	*Decoder
}

func (ed *encoderDecoder) Close() error {
	return ed.Encoder.w.(io.Closer).Close()
}

func New(rw io.ReadWriteCloser, opts ...Option) *Client {
	c := &Client{
		rw: newEncoderDecoder(rw),

		outc: make(chan *out),
		done: make(chan struct{}),

		connackc:  make(chan *packet.Connack),
		pingrespc: make(chan *packet.Pingresp),
		subackc:   make(chan *packet.Suback),
		unsubackc: make(chan *packet.Unsuback),
		pubackc:   make(chan *packet.Puback),
		pubrecc:   make(chan *packet.Pubrec),
		pubcompc:  make(chan *packet.Pubcomp),
	}
	for _, opt := range opts {
		opt(c)
	}
	go c.in()
	go c.out()
	return c
}

type HandlerFunc func(publish *packet.Publish)

type Client struct {
	rw *encoderDecoder

	outc    chan *out
	done    chan struct{}
	err     error
	handler HandlerFunc
	pid     uint16

	warn  Logger
	debug Logger

	connackc  chan *packet.Connack
	pingrespc chan *packet.Pingresp
	subackc   chan *packet.Suback
	unsubackc chan *packet.Unsuback
	pubackc   chan *packet.Puback
	pubrecc   chan *packet.Pubrec
	pubcompc  chan *packet.Pubcomp
}

type out struct {
	pk   packet.OutgoingPacket
	done chan struct{}
}

type Logger interface {
	Print(v ...interface{})
}

func (c *Client) Connect(ctx context.Context, connect *packet.Connect) (*packet.Connack, error) {
	if err := c.send(ctx, connect); err != nil {
		return nil, err
	}
	select {
	case connack := <-c.connackc:
		if connack.ReturnCode != packet.ConnectionAccepted {
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
	if err := c.send(ctx, packet.NewPingreq()); err != nil {
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
	return c.send(ctx, packet.NewDisconnect())
}

var errInvalidPacketID = errors.New("invalid packet id")

func (c *Client) genid() uint16 {
	c.pid++
	return c.pid
}

func (c *Client) Publish(ctx context.Context, publish *packet.Publish) error {
	if publish.PacketID == 0 {
		publish.PacketID = c.genid()
	}
	if err := c.send(ctx, publish); err != nil {
		return err
	}
	switch {
	case enabled(publish.Flags, packet.PublishQoS1):
		select {
		case puback := <-c.pubackc:
			if puback.PacketID != publish.PacketID {
				return errInvalidPacketID
			}
			return nil
		case <-c.done:
			return c.err
		}
	case enabled(publish.Flags, packet.PublishQoS2):
		select {
		case pubrec := <-c.pubrecc:
			if pubrec.PacketID != publish.PacketID {
				return errInvalidPacketID
			}
			if err := c.send(ctx, packet.NewPubrel(publish.PacketID)); err != nil {
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

func enabled(flags packet.Flags, flag uint8) bool {
	return uint8(flags)&flag != 0
}

func (c *Client) Subscribe(
	ctx context.Context, subscribe *packet.Subscribe,
) (*packet.Suback, error) {
	if subscribe.PacketID == 0 {
		subscribe.PacketID = c.genid()
	}
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

func (c *Client) Unsubscribe(ctx context.Context, unsubscribe *packet.Unsubscribe) error {
	if unsubscribe.PacketID == 0 {
		unsubscribe.PacketID = c.genid()
	}
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
		pk, err := c.rw.Decode()
		if err != nil {
			c.close(err)
			return
		}
		c.debugf("< %s", pk)
		switch v := pk.(type) {
		case *packet.Connack:
			select {
			case c.connackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Pingresp:
			select {
			case c.pingrespc <- v:
			case <-c.done:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Puback:
			select {
			case c.pubackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Pubrec:
			select {
			case c.pubrecc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Pubcomp:
			select {
			case c.pubcompc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Suback:
			select {
			case c.subackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Unsuback:
			select {
			case c.unsubackc <- v:
			case <-c.done:
			default:
				c.warnf("unexpected: %s", v)
			}
		case *packet.Publish:
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
		if err := c.rw.Encode(pk.pk); err != nil {
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

func (c *Client) send(ctx context.Context, pk packet.OutgoingPacket) error {
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
	return c.rw.Close()
}
