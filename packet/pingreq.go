package packet

func NewPingreq() *Pingreq {
	return &Pingreq{
		Flags: pkPingreq,
	}
}

type Pingreq struct {
	Flags
}

func (pk *Pingreq) Encode(e Encoder) error {
	return nil
}

func (pk *Pingreq) String() string {
	return "PINGREQ"
}
