package mqtt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

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

type MessagesHandler func(publish *packet.Publish)

func WithMessagesHandler(handler MessagesHandler) Option {
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
	go c.rx()
	go c.tx()
	return c
}

type Client struct {
	mu sync.Mutex
	rw *encoderDecoder

	pkid    uint32
	outc    chan *out
	done    chan struct{}
	err     error
	handler MessagesHandler

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

func (c *Client) genID() uint16 {
	return uint16(atomic.AddUint32(&c.pkid, 1))
}

func (c *Client) Connect(
	ctx context.Context, opts ...packet.ConnectOption,
) (*packet.Connack, error) {
	connect := packet.NewConnect(opts...)
	if err := c.send(ctx, connect); err != nil {
		return nil, err
	}
	select {
	case connack := <-c.connackc:
		if connack.ReturnCode != packet.ConnectionAccepted {
			return nil, fmt.Errorf("connection failed: %s (%d)",
				connack.ReturnCode.String(), connack.ReturnCode)
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

var (
	errInvalidPacketID = errors.New("invalid packet id")
)

func (c *Client) Publish(
	ctx context.Context, topic string, opts ...packet.PublishOption,
) error {
	publish := packet.NewPublish(topic, opts...)
	qos1 := enabled(publish.Flags, packet.PublishQoS1)
	qos2 := enabled(publish.Flags, packet.PublishQoS2)
	if (!qos1 && !qos2) && publish.PacketID != 0 {
		return errors.New("non-zero packet-id for QoS0")
	} else if (qos1 || qos2) && publish.PacketID == 0 {
		publish.PacketID = c.genID()
	}

	if err := c.send(ctx, publish); err != nil {
		return err
	}
	switch {
	case qos1:
		select {
		case puback := <-c.pubackc:
			if puback.PacketID != publish.PacketID {
				return errInvalidPacketID
			}
			return nil
		case <-c.done:
			return c.err
		case <-ctx.Done():
			return ctx.Err()
		}
	case qos2:
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
			case <-ctx.Done():
				return ctx.Err()
			}
		case <-c.done:
			return c.err
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		return nil
	}
}

func enabled(flags packet.Flags, flag uint8) bool {
	return uint8(flags)&flag != 0
}

func (c *Client) Subscribe(
	ctx context.Context, opts ...packet.SubscribeOption,
) (*packet.Suback, error) {
	subscribe := packet.NewSubscribe(opts...)
	if subscribe.PacketID == 0 {
		subscribe.PacketID = c.genID()
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
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Unsubscribe(
	ctx context.Context, opts ...packet.UnsubscribeOption,
) error {
	unsubscribe := packet.NewUnsubscribe(opts...)
	if unsubscribe.PacketID == 0 {
		unsubscribe.PacketID = c.genID()
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
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Client) rx() {
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

func (c *Client) tx() {
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
		select {
		case <-o.done:
			return nil
		case <-c.done:
			return c.err
		case <-ctx.Done():
			return ctx.Err()
		}
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
