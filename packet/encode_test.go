package packet

import (
	"io/ioutil"
	"testing"
)

func BenchmarkEncoder_Encode(b *testing.B) {
	e := NewEncoder(ioutil.Discard)
	p := NewConnect(
		WithConnectClientID("admin"),
		WithConnectUsername("admin"),
		WithConnectPassword("admin"),
		WithConnectWill("bye", []byte("bye"), QoS1, false),
	)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := e.Encode(p); err != nil {
			b.Fatal(err)
		}
	}
}
