package packet

import (
	"fmt"
)

type PubrelOption func(pk *Pubrel)

func WithPubrelPacketID(id uint16) PubrelOption {
	return func(pk *Pubrel) {
		pk.PacketID = id
	}
}

func NewPubrel(packetID uint16, opts ...PubrelOption) *Pubrel {
	pk := &Pubrel{
		Flags:    pkPubrel | 0x02,
		PacketID: packetID,
	}
	for _, opt := range opts {
		opt(pk)
	}
	return pk
}

type Pubrel struct {
	Flags
	PacketID uint16
}

func (pk *Pubrel) encode(e *encoder) error {
	return e.Integer(pk.PacketID)
}

func (pk *Pubrel) String() string {
	return fmt.Sprintf("PUBREL (m%d)", pk.PacketID)
}
