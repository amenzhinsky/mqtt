package mqtt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/amenzhinsky/mqtt/packet"
)

func TestPing(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	if _, err := c.Connect(
		context.Background(),
		packet.WithConnectKeepAlive(1),
		packet.WithConnectCleanSession(true),
	); err != nil {
		t.Fatal(err)
	}

	time.Sleep(500 * time.Millisecond)
	if err := c.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	if err := c.Ping(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("want EOF on expired keep alive, got %v", err)
	}
}

func TestPubSub(t *testing.T) {
	pbc := make(chan *packet.Publish)
	sub := newClient(t, WithMessagesHandler(func(publish *packet.Publish) {
		pbc <- publish
	}))
	defer sub.Close()
	if _, err := sub.Connect(
		context.Background(),
		packet.WithConnectCleanSession(true),
	); err != nil {
		t.Fatal(err)
	}

	if _, err := sub.Subscribe(context.Background(),
		packet.WithSubscribePacketID(666),
		packet.WithSubscribeTopic("test/#", packet.QoS1),
	); err != nil {
		t.Fatal(err)
	}

	pub := newClient(t)
	defer pub.Close()
	if _, err := pub.Connect(
		context.Background(),
		packet.WithConnectCleanSession(true),
	); err != nil {
		t.Fatal(err)
	}

	for _, qos := range []packet.QoS{packet.QoS0, packet.QoS1, packet.QoS2} {
		if err := pub.Publish(context.Background(),
			fmt.Sprintf("test/%d", qos),
			packet.WithPublishQoS(qos),
			packet.WithPublishPayload([]byte{byte(qos)}),
		); err != nil {
			t.Fatal(err)
		}

		select {
		case p := <-pbc:
			if len(p.Payload) != 1 {
				t.Fatal("invalid payload length")
			}
			if packet.QoS(p.Payload[0]) != qos {
				t.Fatalf("qos = %d, want %d", p.Payload[0], qos)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("recv timed out")
		}
	}

	if err := sub.Unsubscribe(context.Background(),
		packet.WithUnsubscribeTopic("test/#"),
	); err != nil {
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
		addr = ":1883"
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return New(conn, opts...)
}
