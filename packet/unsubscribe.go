package packet

import (
	"fmt"
	"strings"
)

type UnsubscribeOption func(pk *Unsubscribe)

func WithUnsubscribePacketID(id uint16) UnsubscribeOption {
	return func(pk *Unsubscribe) {
		pk.PacketID = id
	}
}

func WithUnsubscribeTopic(topics ...string) UnsubscribeOption {
	return func(pk *Unsubscribe) {
		pk.Topics = append(pk.Topics, topics...)
	}
}

func NewUnsubscribe(opts ...UnsubscribeOption) *Unsubscribe {
	pk := &Unsubscribe{
		Flags: pkUnsubscribe | 0x02,
	}
	for _, opt := range opts {
		opt(pk)
	}
	return pk
}

type Unsubscribe struct {
	Flags
	PacketID uint16
	Topics   []string
}

func (pk *Unsubscribe) Encode(e Encoder) error {
	n := integerLen // PacketID
	for _, topic := range pk.Topics {
		n += stringLen(topic)
	}

	var err error
	if err = e.Len(n); err != nil {
		return err
	}
	if err = e.Integer(pk.PacketID); err != nil {
		return err
	}
	for _, topic := range pk.Topics {
		if err = e.String(topic); err != nil {
			return err
		}
	}
	return nil
}

func (pk *Unsubscribe) String() string {
	topics := make([]string, 0, len(pk.Topics))
	for _, topic := range pk.Topics {
		topics = append(topics, topic)
	}
	return fmt.Sprintf("UNSUBSCRIBE (m%d, (%s))", pk.PacketID, strings.Join(topics, ", "))
}
