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
	return e.Len(0)
}

func (pk *Pingreq) String() string {
	return "PINGREQ"
}
