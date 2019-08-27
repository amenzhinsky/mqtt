package packet

import (
	"fmt"
	"strings"
)

type SubscribeReturnCode uint8

const (
	SubscriptionMaxQoS0 = iota
	SubscriptionMaxQoS1
	SubscriptionMaxQoS2
	SubscriptionFailure = 0x80
)

type Suback struct {
	Flags
	PacketID    uint16
	ReturnCodes []uint8
}

func (pk *Suback) Decode(d Decoder) error {
	var err error
	pk.PacketID, err = d.Integer()
	if err != nil {
		return err
	}
	pk.ReturnCodes, err = d.Payload()
	if err != nil {
		return err
	}
	return nil
}

func (pk *Suback) String() string {
	rcs := make([]string, 0, len(pk.ReturnCodes))
	for _, q := range pk.ReturnCodes {
		rcs = append(rcs, fmt.Sprintf("c%d", q))
	}
	return fmt.Sprintf("SUBACK (m%d, (%s))", pk.PacketID, strings.Join(rcs, ", "))
}
