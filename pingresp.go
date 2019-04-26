package mqtt

type Pingresp struct {
	Flags
}

func (pk *Pingresp) decode(d *decoder) error {
	return nil
}

func (pk *Pingresp) String() string {
	return "PINGRESP"
}
