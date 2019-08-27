package packet

import (
	"fmt"
)

const (
	AcknowledgeSessionPresent = 1
)

type ConnectReturnCode uint8

const (
	ConnectionAccepted ConnectReturnCode = iota
	ConnectionUnacceptableProtocolVersion
	ConnectionIdentifierRejected
	ConnectionServerUnavailable
	ConnectionBadUsernameOrPassword
	ConnectionNotAuthorized
)

func (rc ConnectReturnCode) String() string {
	switch rc {
	case ConnectionAccepted:
		return "accepted"
	case ConnectionUnacceptableProtocolVersion:
		return "unacceptable protocol version"
	case ConnectionIdentifierRejected:
		return "identifier rejected"
	case ConnectionServerUnavailable:
		return "server unavailable"
	case ConnectionBadUsernameOrPassword:
		return "bad user name or password"
	case ConnectionNotAuthorized:
		return "not authorized"
	default:
		return fmt.Sprintf("unknown(%d)", rc)
	}
}

type ConnackOption func(pk *Connack)

func WithConnackSessionPresent(enabled bool) ConnackOption {
	return func(pk *Connack) {
		if enabled {
			pk.AcknowledgeFlags |= AcknowledgeSessionPresent
		}
	}
}

func WithConnackReturnCode(rc ConnectReturnCode) ConnackOption {
	return func(pk *Connack) {
		pk.ReturnCode = rc
	}
}

func NewConnack(opts ...ConnackOption) *Connack {
	pk := &Connack{
		Flags: pkConnack,
	}
	for _, opt := range opts {
		opt(pk)
	}
	return pk
}

type Connack struct {
	Flags
	AcknowledgeFlags uint8
	ReturnCode       ConnectReturnCode
}

func (pk *Connack) Decode(d Decoder) error {
	var err error
	pk.AcknowledgeFlags, err = d.Bits()
	if err != nil {
		return err
	}
	rc, err := d.Bits()
	if err != nil {
		return err
	}
	pk.ReturnCode = ConnectReturnCode(rc)
	return nil
}

func (pk Connack) String() string {
	return fmt.Sprintf("CONNACK (c%d, s%d)",
		pk.ReturnCode,
		flag(pk.AcknowledgeFlags, AcknowledgeSessionPresent),
	)
}
