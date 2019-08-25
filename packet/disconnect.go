package packet

func NewDisconnect() *Disconnect {
	return &Disconnect{
		Flags: pkDisconnect,
	}
}

type Disconnect struct {
	Flags
}

func (pk *Disconnect) encode(e *encoder) error {
	return nil
}

func (pk *Disconnect) String() string {
	return "DISCONNECT"
}
