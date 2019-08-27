package mqtt

import (
	"io/ioutil"
	"testing"

	"github.com/amenzhinsky/mqtt/packet"
)

func BenchmarkEncoder_Encode(b *testing.B) {
	e := NewEncoder(ioutil.Discard)
	p := packet.NewConnect(
		packet.WithConnectClientID("admin"),
		packet.WithConnectUsername("admin"),
		packet.WithConnectPassword("admin"),
		packet.WithConnectWill("bye", []byte("bye"), packet.QoS1, false),
	)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := e.Encode(p); err != nil {
			b.Fatal(err)
		}
	}
}
