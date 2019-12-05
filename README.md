# mqtt

Micro [MQTT 3.1.1](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) client for [Golang](https://golang.org/).

It provides low-level API and doesn't support async usage and advanced features such as reconnects, fallback addresses and persistent storage.

## Usage

```go
conn, err := net.Dial("tcp", "127.0.0.1:8883")
if err != nil {
	return err
}

client := mqtt.New(conn, mqtt.WithMessagesHandler(func(pk *packet.Publish) {
	switch {
	case mqtt.Match("/dev/cmd/speed", pk.Topic):
		speed(binary.BigEndian.Uint32(pk.Payload))
	case mqtt.Match("/dev/cmd/+/play", pk.Topic):
		play(strings.Split(pk.Topic, "/")[3], pk.Payload)
	}
}))
defer client.Close()

if _, err := client.Connect(context.Background(), packet.NewConnect(
	packet.WithConnectCleanSession(true),
	packet.WithConnectWill("/dev/online", []byte{0}, packet.QoS1, true),
)); err != nil {
	return err
}

if _, err := client.Subscribe(context.Background(), packet.NewSubscribe(
	packet.WithSubscribePacketID(1),
	packet.WithSubscribeTopic("/dev/cmd/#", packet.QoS1),
)); err != nil {
	return err
}

if err := client.Publish(context.Background(), packet.NewPublish(
	"/dev/online",
	packet.WithPublishQoS(packet.QoS1),
	packet.WithPublishPayload([]byte{1}),
)); err != nil {
	return err
}

if err := client.Disconnect(context.Background()); err != nil {
	return err
}
```
