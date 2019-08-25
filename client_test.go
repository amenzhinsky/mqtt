package mqtt

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/amenzhinsky/mqtt/packet"
)

func TestPubSub(t *testing.T) {
	pbc := make(chan *packet.Publish)
	sub := newClient(t, WithHandler(func(publish *packet.Publish) {
		pbc <- publish
	}))
	defer sub.Close()
	if _, err := sub.Connect(
		context.Background(),
		packet.NewConnect(
			packet.WithConnectCleanSession(true),
		),
	); err != nil {
		t.Fatal(err)
	}

	if _, err := sub.Subscribe(context.Background(), packet.NewSubscribe(
		packet.WithSubscribeTopic("test/#", packet.QoS1),
	)); err != nil {
		t.Fatal(err)
	}

	pub := newClient(t)
	defer pub.Close()
	if _, err := pub.Connect(
		context.Background(),
		packet.NewConnect(
			packet.WithConnectCleanSession(true),
		),
	); err != nil {
		t.Fatal(err)
	}

	// TODO: separate test
	if err := pub.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, qos := range []packet.QoS{packet.QoS0, packet.QoS1, packet.QoS2} {
		if err := pub.Publish(context.Background(),
			packet.NewPublish(fmt.Sprintf("test/%d", qos),
				packet.WithPublishQoS(packet.QoS0),
				packet.WithPublishPayload([]byte{byte(qos)}),
			),
		); err != nil {
			t.Fatal(err)
		}

		select {
		case p := <-pbc:
			if packet.QoS(p.Payload[0]) != qos {
				t.Fatalf("invalid QoS have %d, want %d", p.Payload[0], qos)
			}
		case <-time.After(time.Second):
			t.Fatal("recv timed out")
		}
	}

	if err := sub.Unsubscribe(context.Background(), packet.NewUnsubscribe(
		packet.WithUnsubscribeTopic("test/#"),
	)); err != nil {
		t.Fatal(err)
	}
	if err := sub.Disconnect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := pub.Disconnect(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func newClient(t *testing.T, opts ...Option) *Client {
	t.Helper()
	addr := os.Getenv("TEST_MQTT_ADDR")
	if addr == "" {
		t.Errorf("TEST_MQTT_ADDR is empty")
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return New(conn, opts...)
}