//+build fuzz

package mqtt

import (
	"bytes"
)

func FuzzEncode(b []byte) int {
	if _, err := NewDecoder(bytes.NewReader(b)).Decode(); err != nil {
		return 0
	}
	return 1
}
