package packet

func NewDisconnect() *Disconnect {
	return &Disconnect{
		Flags: pkDisconnect,
	}
}

type Disconnect struct {
	Flags
}

func (pk *Disconnect) Encode(e Encoder) error {
	return e.Len(0)
}

func (pk *Disconnect) String() string {
	return "DISCONNECT"
}
