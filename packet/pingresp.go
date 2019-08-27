package packet

type Pingresp struct {
	Flags
}

func (pk *Pingresp) Decode(d Decoder) error {
	return nil
}

func (pk *Pingresp) String() string {
	return "PINGRESP"
}
