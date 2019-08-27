package packet

import (
	"fmt"
	"strings"
)

type SubscribeOption func(pk *Subscribe)

func WithSubscribePacketID(id uint16) SubscribeOption {
	return func(pk *Subscribe) {
		pk.PacketID = id
	}
}

func WithSubscribeTopic(topic string, qos QoS) SubscribeOption {
	return func(pk *Subscribe) {
		pk.Topics = append(pk.Topics, &SubscribeTopic{
			Name:  topic,
			Flags: uint8(qos),
		})
	}
}

func NewSubscribe(opts ...SubscribeOption) *Subscribe {
	pk := &Subscribe{
		Flags: pkSubscribe | 0x02,
	}
	for _, opt := range opts {
		opt(pk)
	}
	return pk
}

type Subscribe struct {
	Flags
	PacketID uint16
	Topics   []*SubscribeTopic
}

type SubscribeTopic struct {
	Name  string
	Flags uint8
}

func (t *SubscribeTopic) String() string {
	return fmt.Sprintf("%s q%d", t.Name, t.Flags)
}

func (pk *Subscribe) Encode(e Encoder) error {
	var err error
	if err = e.Integer(pk.PacketID); err != nil {
		return err
	}
	for _, topic := range pk.Topics {
		if err = e.String(topic.Name); err != nil {
			return err
		}
		if err = e.Bits(topic.Flags); err != nil {
			return err
		}
	}
	return nil
}

func (pk *Subscribe) String() string {
	topics := make([]string, 0, len(pk.Topics))
	for _, topic := range pk.Topics {
		topics = append(topics, topic.String())
	}
	return fmt.Sprintf("SUBSCRIBE (m%d, (%s))", pk.PacketID, strings.Join(topics, ", "))
}
