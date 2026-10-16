package proxy

import (
	"strings"
	"testing"
)

func TestClientIdentityBoundaries(t *testing.T) {
	for _, v := range []string{"", "client/1.0 (windows)", strings.Repeat("x", 1024)} {
		if !ValidClientUserAgent(v) {
			t.Fatal("valid identity rejected")
		}
	}
	for _, v := range []string{"client\r\nAuthorization: private", "client\x00", "非ASCII", strings.Repeat("x", 1025)} {
		if ValidClientUserAgent(v) {
			t.Fatal("unsafe identity accepted")
		}
	}
}
