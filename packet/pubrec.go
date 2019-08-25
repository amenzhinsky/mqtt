package packet

import (
	"fmt"
)

type Pubrec struct {
	Flags
	PacketID uint16
}

func (pk *Pubrec) decode(d *decoder) error {
	var err error
	pk.PacketID, err = d.Integer()
	if err != nil {
		return err
	}
	return nil
}

func (pk *Pubrec) String() string {
	return fmt.Sprintf("PUBREC (m%d)", pk.PacketID)
}
