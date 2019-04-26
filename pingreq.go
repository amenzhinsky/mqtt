package mqtt

import "io"

func NewPingreqPacket() *Pingreq {
	return &Pingreq{
		Flags: pkPingreq,
	}
}

type Pingreq struct {
	Flags
}

func (pk *Pingreq) encode(w io.Writer) error {
	return nil
}

func (pk *Pingreq) String() string {
	return "PINGREQ"
}
