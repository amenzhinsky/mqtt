package mqtt

import (
	"fmt"
)

var ProtocolName = "MQTT"

const ProtocolLevel311 = 0x04

const (
	ConnectFlagCleanSession = 2 << iota
	ConnectFlagWillFlag
	ConnectFlagWillQoS1
	ConnectFlagWillQoS2
	ConnectFlagWillRetain
	ConnectFlagPassword
	ConnectFlagUsername
)

type ConnectOption func(pk *Connect)

func WithConnectProtocolName(name string) ConnectOption {
	return func(pk *Connect) {
		pk.ProtocolName = name
	}
}

func WithConnectProtocolLevel(level uint8) ConnectOption {
	return func(pk *Connect) {
		pk.ProtocolLevel = level
	}
}

func WithConnectKeepAlive(sec uint16) ConnectOption {
	return func(pk *Connect) {
		pk.KeepAlive = sec
	}
}

func WithConnectClientID(id string) ConnectOption {
	return func(pk *Connect) {
		pk.ClientID = id
	}
}

func WithConnectCleanSession(enabled bool) ConnectOption {
	return func(pk *Connect) {
		if enabled {
			pk.ConnectFlags |= ConnectFlagCleanSession
		}
	}
}

func WithConnectWill(topic string, payload []byte, qos QoS, retained bool) ConnectOption {
	return func(pk *Connect) {
		pk.WillTopic = topic
		pk.WillPayload = payload
		pk.ConnectFlags |= ConnectFlagWillFlag
		switch qos {
		case QoS0:
		case QoS1:
			pk.ConnectFlags |= ConnectFlagWillQoS1
		case QoS2:
			pk.ConnectFlags |= ConnectFlagWillQoS2
		default:
			panic(fmt.Sprintf("unknown QoS level %d", qos))
		}
		if retained {
			pk.ConnectFlags |= ConnectFlagWillRetain
		}
	}
}

func WithConnectUsername(username string) ConnectOption {
	return func(pk *Connect) {
		pk.Username = username
		if username != "" {
			pk.ConnectFlags |= ConnectFlagUsername
		}
	}
}

func WithConnectPassword(password string) ConnectOption {
	return func(pk *Connect) {
		pk.Password = password
		if password != "" {
			pk.ConnectFlags |= ConnectFlagPassword
		}
	}
}

func NewConnectPacket(opts ...ConnectOption) *Connect {
	pk := &Connect{
		Flags:         pkConnect,
		ProtocolName:  ProtocolName,
		ProtocolLevel: ProtocolLevel311,
	}
	for _, opt := range opts {
		opt(pk)
	}
	return pk
}

type Connect struct {
	Flags
	ProtocolName  string
	ProtocolLevel uint8
	ConnectFlags  uint8
	KeepAlive     uint16
	ClientID      string
	WillTopic     string
	WillPayload   []byte
	Username      string
	Password      string
}

func (pk *Connect) encode(e *encoder) error {
	var err error
	if err = e.String(pk.ProtocolName); err != nil {
		return err
	}
	if err = e.Bits(pk.ProtocolLevel); err != nil {
		return err
	}
	if err = e.Bits(pk.ConnectFlags); err != nil {
		return err
	}
	if err = e.Integer(pk.KeepAlive); err != nil {
		return err
	}
	if err = e.String(pk.ClientID); err != nil {
		return err
	}
	if enabled(pk.ConnectFlags, ConnectFlagWillFlag) {
		if err = e.String(pk.WillTopic); err != nil {
			return err
		}
		if err = e.Bytes(pk.WillPayload); err != nil {
			return err
		}
	}
	if enabled(pk.ConnectFlags, ConnectFlagUsername) {
		if err = e.String(pk.Username); err != nil {
			return err
		}
	}
	if enabled(pk.ConnectFlags, ConnectFlagPassword) {
		if err = e.String(pk.Password); err != nil {
			return err
		}
	}
	return nil
}

func (pk *Connect) String() string {
	// TODO: add will props
	return fmt.Sprintf("CONNECT (p%d, c%d, k%d, i%q)",
		pk.ProtocolLevel,
		flag(pk.ConnectFlags, ConnectFlagCleanSession),
		pk.KeepAlive,
		pk.ClientID,
	)
}
