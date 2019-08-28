package packet

import (
	"fmt"
)

type Flags uint8

func (h Flags) GetFlags() Flags {
	return h
}

type QoS byte

const (
	QoS0 = iota
	QoS1
	QoS2
)

const (
	pkConnect = (iota + 1) << 4
	pkConnack
	pkPublish
	pkPuback
	pkPubrec
	pkPubrel
	pkPubcomp
	pkSubscribe
	pkSuback
	pkUnsubscribe
	pkUnsuback
	pkPingreq
	pkPingresp
	pkDisconnect
)

type Encoder interface {
	Len(int) error
	Bits(uint8) error
	Integer(uint16) error
	Payload([]byte) error
	Bytes([]byte) error
	String(string) error
}

type Decoder interface {
	Bits() (uint8, error)
	Integer() (uint16, error)
	Payload() ([]byte, error)
	Bytes() ([]byte, error)
	String() (string, error)
}

func NewIncomingPacket(fh uint8) IncomingPacket {
	switch fh & 0xf0 {
	case pkConnack:
		return &Connack{Flags: Flags(fh)}
	case pkPublish:
		return &Publish{Flags: Flags(fh)}
	case pkPuback:
		return &Puback{Flags: Flags(fh)}
	case pkPubrec:
		return &Pubrec{Flags: Flags(fh)}
	case pkPubcomp:
		return &Pubcomp{Flags: Flags(fh)}
	case pkSuback:
		return &Suback{Flags: Flags(fh)}
	case pkUnsuback:
		return &Unsuback{Flags: Flags(fh)}
	case pkPingresp:
		return &Pingresp{Flags: Flags(fh)}
	default:
		return nil
	}
}

type OutgoingPacket interface {
	packet
	Encode(e Encoder) error
}

type IncomingPacket interface {
	packet
	Decode(d Decoder) error
}

type packet interface {
	fmt.Stringer
	GetFlags() Flags
}

func flag(flags, flag uint8) uint8 {
	if enabled(flags, flag) {
		return 1
	}
	return 0
}

func enabled(flags, flag uint8) bool {
	return flags&flag != 0
}

const (
	bitsLen    = 1
	integerLen = 2
)

func stringLen(s string) int {
	return integerLen + len(s)
}

func bytesLen(b []byte) int {
	return integerLen + len(b)
}
