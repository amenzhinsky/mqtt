package mqtt

func NewPingreqPacket() *Pingreq {
	return &Pingreq{
		Flags: pkPingreq,
	}
}

type Pingreq struct {
	Flags
}

func (pk *Pingreq) encode(e *encoder) error {
	return nil
}

func (pk *Pingreq) String() string {
	return "PINGREQ"
}
