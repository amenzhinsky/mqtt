package packet

import (
	"fmt"
)

//type Packet struct {
//	Flags
//}
//
//func ReadFrom(r io.Reader) error {
//	fh := make([]byte, 2)
//	_, err := r.Read(fh)
//	if err != nil {
//		return err
//	}
//
//	pk := &Packet{Flags: Flags(fh[0])}
//
//}
//
//func readByte(r io.Reader) (byte, error) {
//	if br, ok := r.(io.ByteReader); ok {
//		return br.ReadByte()
//	}
//	b := make([]byte, 1)
//	if _, err := r.Read(b); err != nil {
//		return 0, err
//	}
//	return b[0], nil
//}
//
//func readBytes(r io.Reader, n int) ([]byte, error) {
//	b := make([]byte, n)
//	if _, err := io.ReadFull(r, b); err != nil {
//		return nil, err
//	}
//	return b, nil
//}

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
