package packet

import (
	"fmt"
)

type Puback struct {
	Flags
	PacketID uint16
}

func (pk *Puback) decode(d *decoder) error {
	var err error
	pk.PacketID, err = d.Integer()
	if err != nil {
		return err
	}
	return nil
}

func (pk *Puback) String() string {
	return fmt.Sprintf("PUBACK (m%d)", pk.PacketID)
}
