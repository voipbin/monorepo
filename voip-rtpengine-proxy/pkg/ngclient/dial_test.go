package ngclient

import (
	"strings"
	"testing"
	"time"
)

// A link-local IPv6 address without a zone cannot be connected on Linux, which
// makes DialUDP fail while the address itself resolves.
func Test_newNGClient_dialError(t *testing.T) {
	c, err := newNGClient("[fe80::1]:1", time.Second)
	if err == nil {
		c.Close()
		t.Fatalf("Wrong match. expected: dial error, got: nil")
	}
	if !strings.Contains(err.Error(), "dial RTPEngine NG") {
		t.Errorf("Wrong match. expected dial error, got: %v", err)
	}
}
