package packet

import (
	"fmt"
)

const (
	PublishRetain = 1 << iota
	PublishQoS1
	PublishQoS2
	PublishDup
)

type PublishOption func(pk *Publish)

func WithPublishRetain(enabled bool) PublishOption {
	return func(pk *Publish) {
		if enabled {
			pk.Flags |= PublishRetain
		}
	}
}

func WithPublishQoS(qos QoS) PublishOption {
	return func(pk *Publish) {
		switch qos {
		case QoS0:
		case QoS1:
			pk.Flags |= PublishQoS1
		case QoS2:
			pk.Flags |= PublishQoS2
		default:
			panic(fmt.Sprintf("unknown QoS level: %d", qos))
		}
	}
}

// TODO: set only by server?
func WithPublishDup(enabled bool) PublishOption {
	return func(pk *Publish) {
		if enabled {
			pk.Flags |= PublishDup
		}
	}
}

func WithPublishPacketID(id uint16) PublishOption {
	return func(pk *Publish) {
		pk.PacketID = id
	}
}

func WithPublishPayload(payload []byte) PublishOption {
	return func(pk *Publish) {
		pk.Payload = payload
	}
}

func NewPublish(topic string, opts ...PublishOption) *Publish {
	pk := &Publish{
		Flags: pkPublish,
		Topic: topic,
	}
	for _, opt := range opts {
		opt(pk)
	}
	return pk
}

type Publish struct {
	Flags
	Topic    string
	Payload  []byte
	PacketID uint16
}

func (pk *Publish) encode(e *encoder) error {
	var err error
	if err = e.String(pk.Topic); err != nil {
		return err
	}
	if enabled(uint8(pk.Flags), PublishQoS1|PublishQoS2) {
		if err = e.Integer(pk.PacketID); err != nil {
			return err
		}
	}
	if pk.Payload != nil {
		return e.Payload(pk.Payload)
	}
	return nil
}

func (pk *Publish) decode(d *decoder) error {
	var err error
	pk.Topic, err = d.String()
	if err != nil {
		return err
	}
	pk.Payload, err = d.Payload()
	if err != nil {
		return err
	}
	return nil
}

func (pk *Publish) String() string {
	var qos uint8
	if enabled(uint8(pk.Flags), PublishQoS1) {
		qos = 1
	} else if enabled(uint8(pk.Flags), PublishQoS2) {
		qos = 2
	}

	return fmt.Sprintf("PUBLISH (d%d, r%d, q%d, m%d, %q, ... (%d bytes))",
		flag(uint8(pk.Flags), PublishDup),
		flag(uint8(pk.Flags), PublishRetain),
		qos,
		pk.PacketID,
		pk.Topic,
		len(pk.Payload),
	)
}
