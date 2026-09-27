package netutil

import (
	"strings"
	"testing"
)

func TestLANURLsFormat(t *testing.T) {
	urls := LANURLs(8080)
	for _, u := range urls {
		if !strings.HasPrefix(u, "http://") {
			t.Fatalf("got URL %q, want an http:// prefix", u)
		}
		if !strings.HasSuffix(u, ":8080") {
			t.Fatalf("got URL %q, want a :8080 suffix", u)
		}
		if strings.Contains(u, "127.0.0.1") {
			t.Fatalf("got loopback address in %q, want only LAN-reachable addresses", u)
		}
	}
}
