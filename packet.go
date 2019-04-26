package mqtt

import (
	"fmt"
)

type Flags uint8

func (h Flags) flags() Flags {
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

type OutgoingPacket interface {
	packet
	encode(e *encoder) error
}

type IncomingPacket interface {
	packet
	decode(d *decoder) error
}

type packet interface {
	fmt.Stringer
	flags() Flags
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
