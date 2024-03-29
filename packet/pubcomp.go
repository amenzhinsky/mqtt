package packet

import (
	"fmt"
)

type Pubcomp struct {
	Flags
	PacketID uint16
}

func (pk *Pubcomp) Decode(d Decoder) error {
	var err error
	pk.PacketID, err = d.Integer()
	if err != nil {
		return err
	}
	return nil
}

func (pk *Pubcomp) String() string {
	return fmt.Sprintf("PUBCOMP (m%d)", pk.PacketID)
}
