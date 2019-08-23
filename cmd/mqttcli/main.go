package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/amenzhinsky/mqtt"
)

var (
	addrFlag         string
	cleanSessionFlag bool
	clientIDFlag     string
	usernameFlag     string
	passwordFlag     string
	keepAliveFlag    uint
	debugFlag        bool

	willTopicFlag   string
	willPayloadFlag string
	willQoSFlag     uint
	willRetainFlag  bool
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [common flags...] [pub|sub] [flag...] [arg...]

Common Flags:
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.StringVar(&addrFlag, "addr", ":1883", "address to connect to")
	flag.BoolVar(&cleanSessionFlag, "clean-session", true, "clean session")
	flag.StringVar(&clientIDFlag, "client-id", "", "client id")
	flag.StringVar(&usernameFlag, "username", "", "username")
	flag.StringVar(&passwordFlag, "password", "", "password")
	flag.UintVar(&keepAliveFlag, "keep-alive", 0, "keep alive")
	flag.BoolVar(&debugFlag, "debug", false, "enable debug mode")
	flag.StringVar(&willTopicFlag, "will-topic", "", "topic name to publish the will")
	flag.StringVar(&willPayloadFlag, "will-payload", "", "payload of the client will")
	flag.UintVar(&willQoSFlag, "will-qos", 0, "QoS level of the will")
	flag.BoolVar(&willRetainFlag, "will-retain", false, "make the will retained")
	flag.Parse()
	if flag.NArg() == 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	switch flag.Arg(0) {
	case "pub":
		return pub(connect, flag.Args()[1:])
	case "sub":
		return sub(connect, flag.Args()[1:])
	default:
		return fmt.Errorf("unknown command %q", flag.Arg(0))
	}
}

func connect() (*mqtt.Client, error) {
	conn, err := net.Dial("tcp", addrFlag)
	if err != nil {
		return nil, err
	}
	c := mqtt.NewClient(conn)

	opts := []mqtt.ConnectOption{
		mqtt.WithConnectCleanSession(cleanSessionFlag),
		mqtt.WithConnectClientID(clientIDFlag),
		mqtt.WithConnectUsername(usernameFlag),
		mqtt.WithConnectPassword(passwordFlag),
		mqtt.WithConnectKeepAlive(uint16(keepAliveFlag)),
	}
	if willTopicFlag != "" {
		opts = append(opts, mqtt.WithConnectWill(
			willTopicFlag, []byte(willPayloadFlag), mqtt.QoS(willQoSFlag), willRetainFlag,
		))
	}
	if err = c.Send(mqtt.NewConnectPacket(opts...)); err != nil {
		return nil, err
	}
	pk, err := c.Recv()
	if err != nil {
		return nil, err
	}
	connack, ok := pk.(*mqtt.Connack)
	if !ok {
		return nil, fmt.Errorf("expected CONNACK packet, got %s", pk.String())
	}
	if connack.ReturnCode != mqtt.ConnectionAccepted {
		return nil, errors.New(connack.ReturnCode.String())
	}
	return c, nil
}

func pub(connect func() (*mqtt.Client, error), argv []string) error {
	var (
		qosFlag      uint
		retainFlag   bool
		packetIDFlag uint
	)

	fset := flag.NewFlagSet("pub", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [common flags...] pub [flags...] TOPIC [payload]

Flags:
`, filepath.Base(os.Args[0]))
		fset.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommon Flags:\n")
		flag.PrintDefaults()
	}
	fset.UintVar(&qosFlag, "qos", 0, "qos level")
	fset.BoolVar(&retainFlag, "retain", false, "retained message")
	fset.UintVar(&packetIDFlag, "packet-id", 1, "packet identifier")
	_ = fset.Parse(argv) // exits on error

	var payload []byte
	switch fset.NArg() {
	case 1:
		// OK
	case 2:
		payload = []byte(fset.Arg(1))
	default:
		fset.Usage()
		os.Exit(2)
	}

	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Send(mqtt.NewPublishPacket(fset.Arg(0),
		mqtt.WithPublishPayload(payload),
		mqtt.WithPublishQoS(mqtt.QoS(qosFlag)),
		mqtt.WithPublishRetain(retainFlag),
		mqtt.WithPublishPacketID(uint16(packetIDFlag)),
	)); err != nil {
		return err
	}

	switch qosFlag {
	case mqtt.QoS1:
		pk, err := c.Recv()
		if err != nil {
			return err
		}
		puback, ok := pk.(*mqtt.Puback)
		if !ok {
			return fmt.Errorf("expected PUBACK packet, got %s", pk.String())
		}
		if puback.PacketID != uint16(packetIDFlag) {
			return fmt.Errorf("unexpected PUBACK id %d, want %d", puback.PacketID, uint16(packetIDFlag))
		}
	case mqtt.QoS2:
		pk, err := c.Recv()
		if err != nil {
			return err
		}

		// part 1
		pubrec, ok := pk.(*mqtt.Pubrec)
		if !ok {
			return fmt.Errorf("expected PUBREC packet, got %s", pk.String())
		}
		if pubrec.PacketID != uint16(packetIDFlag) {
			return fmt.Errorf("unexpected PUBREC id %d, want %d", pubrec.PacketID, uint16(packetIDFlag))
		}

		// part 2
		if err = c.Send(mqtt.NewPubrelPacket(uint16(packetIDFlag))); err != nil {
			return err
		}

		// part 3
		pk, err = c.Recv()
		if err != nil {
			return err
		}
		pubcomp, ok := pk.(*mqtt.Pubcomp)
		if !ok {
			return fmt.Errorf("expected PUBCOMP packet, got %s", pk.String())
		}
		if pubcomp.PacketID != uint16(packetIDFlag) {
			return fmt.Errorf("unexpected PUBCOMP id %d, want %d", pubcomp.PacketID, uint16(packetIDFlag))
		}
	}
	return c.Send(mqtt.NewDisconnectPacket())
}

func sub(connect func() (*mqtt.Client, error), argv []string) error {
	var (
		qosFlag      uint
		packetIDFlag uint
	)

	fset := flag.NewFlagSet("pub", flag.ExitOnError)
	fset.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [common flags...] sub [flags...] TOPIC...

Flags:
`, filepath.Base(os.Args[0]))
		fset.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommon Flags:\n")
		flag.PrintDefaults()
	}
	fset.UintVar(&qosFlag, "qos", 0, "qos level")
	fset.UintVar(&packetIDFlag, "packet-id", 1, "packet identifier")
	_ = fset.Parse(argv) // exits on error
	if fset.NArg() == 0 {
		fset.Usage()
		os.Exit(2)
	}

	c, err := connect()
	if err != nil {
		return err
	}
	defer c.Close()

	opts := make([]mqtt.SubscribeOption, 0, fset.NArg()+1)
	opts = append(opts, mqtt.WithSubscribePacketID(uint16(packetIDFlag)))
	for _, topic := range fset.Args() {
		opts = append(opts, mqtt.WithSubscribeTopic(topic, mqtt.QoS(qosFlag)))
	}
	if err = c.Send(mqtt.NewSubscribePacket(opts...)); err != nil {
		return err
	}

	pk, err := c.Recv()
	if err != nil {
		return err
	}
	connack, ok := pk.(*mqtt.Suback)
	if !ok {
		return fmt.Errorf("expected SUBACK packet, got %s", pk.String())
	}
	_ = connack

	for {
		pk, err := c.Recv()
		if err != nil {
			return err
		}
		publish, ok := pk.(*mqtt.Publish)
		if !ok {
			return fmt.Errorf("expected PUBLISH packet, got %s", pk.String())
		}
		fmt.Printf("%s %s\n", publish.Topic, publish.Payload)
	}

	return c.Send(mqtt.NewDisconnectPacket())
}
