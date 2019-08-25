package packet

import (
	"fmt"
)

type Unsuback struct {
	Flags
	PacketID uint16
}

func (pk *Unsuback) decode(d *decoder) error {
	var err error
	pk.PacketID, err = d.Integer()
	if err != nil {
		return err
	}
	return nil
}

func (pk *Unsuback) String() string {
	return fmt.Sprintf("UNSUBACK (m%d)", pk.PacketID)
}
