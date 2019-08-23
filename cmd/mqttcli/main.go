package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	go func() {
		<-sigc
		cancel()
		<-sigc
		os.Exit(1)
	}()

	switch flag.Arg(0) {
	case "pub":
		return pub(ctx, connect, flag.Args()[1:])
	case "sub":
		return sub(ctx, connect, flag.Args()[1:])
	default:
		return fmt.Errorf("unknown command %q", flag.Arg(0))
	}
}

func connect(ctx context.Context, opts ...mqtt.Option) (*mqtt.Client, error) {
	conn, err := net.Dial("tcp", addrFlag)
	if err != nil {
		return nil, err
	}
	c := mqtt.NewClient(conn, append([]mqtt.Option{
		mqtt.WithWarnLogger(log.New(os.Stderr, "W ", 0)),
		mqtt.WithDebugLogger(log.New(os.Stderr, "D ", 0)),
	}, opts...)...)

	copts := []mqtt.ConnectOption{
		mqtt.WithConnectCleanSession(cleanSessionFlag),
		mqtt.WithConnectClientID(clientIDFlag),
		mqtt.WithConnectUsername(usernameFlag),
		mqtt.WithConnectPassword(passwordFlag),
		mqtt.WithConnectKeepAlive(uint16(keepAliveFlag)),
	}
	if willTopicFlag != "" {
		copts = append(copts, mqtt.WithConnectWill(
			willTopicFlag, []byte(willPayloadFlag), mqtt.QoS(willQoSFlag), willRetainFlag,
		))
	}
	if _, err = c.Connect(ctx, mqtt.NewConnectPacket(copts...)); err != nil {
		return nil, err
	}
	return c, nil
}

type connectFunc func(ctx context.Context, opts ...mqtt.Option) (*mqtt.Client, error)

func pub(ctx context.Context, connect connectFunc, argv []string) error {
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

	c, err := connect(ctx)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.Publish(ctx, mqtt.NewPublishPacket(fset.Arg(0),
		mqtt.WithPublishPayload(payload),
		mqtt.WithPublishQoS(mqtt.QoS(qosFlag)),
		mqtt.WithPublishRetain(retainFlag),
		mqtt.WithPublishPacketID(uint16(packetIDFlag)),
	)); err != nil {
		return err
	}
	return c.Disconnect(ctx)
}

func sub(ctx context.Context, connect connectFunc, argv []string) error {
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

	c, err := connect(ctx, mqtt.WithHandler(func(publish *mqtt.Publish) {
		fmt.Printf("%s %s\n", publish.Topic, string(publish.Payload))
	}))
	if err != nil {
		return err
	}
	defer c.Close()

	opts := make([]mqtt.SubscribeOption, 0, fset.NArg()+1)
	opts = append(opts, mqtt.WithSubscribePacketID(uint16(packetIDFlag)))
	for _, topic := range fset.Args() {
		opts = append(opts, mqtt.WithSubscribeTopic(topic, mqtt.QoS(qosFlag)))
	}
	if _, err = c.Subscribe(ctx, mqtt.NewSubscribePacket(opts...)); err != nil {
		return err
	}

	<-ctx.Done()
	return c.Disconnect(context.Background())
}
