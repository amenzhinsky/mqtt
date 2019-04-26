package mqtt

import (
	"bytes"
	"io"
	"testing"
)

func TestRingBuffer(t *testing.T) {
	buf := buffer{r: bytes.NewBufferString("12345")}
	c, err := buf.Byte()
	if err != nil {
		t.Fatal(err)
	}
	if c != '1' {
		t.Fatalf("byte = %q, want %q", c, '1')
	}
	v, err := buf.Bytes(4)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte("2345"); !bytes.Equal(v, want) {
		t.Fatalf("bytes = %v, want %v", v, want)
	}
	if _, err = buf.Byte(); err != nil && err.Error() != "EOF" {
		t.Fatalf("read err = %v, want %v", err, io.EOF)
	}
}

func BenchmarkDecoder_Decode(b *testing.B) {
	d := NewDecoder(&loopReader{buf: []byte{
		0x30, 0x11, 0x0, 0x8, 0x70, 0x75, 0x62, 0x2f,
		0x74, 0x65, 0x73, 0x74, 0x70, 0x61, 0x79, 0x6c,
		0x6f, 0x61, 0x64,
	}})
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := d.Decode(); err != nil {
			b.Fatal(err)
		}
	}
}

type loopReader struct {
	buf []byte
	off int
}

func (r *loopReader) Read(b []byte) (int, error) {
	var tlen int
	for {
		clen := copy(b[tlen:], r.buf[r.off:])
		if clen == 0 {
			panic("clen is zero")
		}
		tlen += clen
		r.off += clen
		if r.off == len(r.buf) {
			r.off = 0
		}
		if tlen >= len(b) {
			return tlen, nil
		}
	}
}
