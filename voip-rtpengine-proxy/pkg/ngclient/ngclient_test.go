package ngclient

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/zeebo/bencode"
)

func TestCookieUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		c := newCookie()
		if seen[c] {
			t.Errorf("duplicate cookie: %q", c)
		}
		seen[c] = true
	}
}

// newTestUDPServer starts a local UDP server. The handler receives each packet
// and returns a response payload (or "" to not respond).
func newTestUDPServer(t *testing.T, handler func(req []byte) []byte) string {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	go func() {
		buf := make([]byte, 65535)
		for {
			n, peer, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			resp := handler(buf[:n])
			if resp != nil {
				_, _ = conn.WriteTo(resp, peer)
			}
		}
	}()
	return conn.LocalAddr().String()
}

// echoHandler parses an NG request and echoes back a success response using
// the same cookie and the provided result dict.
func echoHandler(result map[string]interface{}) func([]byte) []byte {
	return func(req []byte) []byte {
		idx := bytes.IndexByte(req, ' ')
		if idx < 0 {
			return nil
		}
		cookie := string(req[:idx])
		encoded, err := bencode.EncodeString(result)
		if err != nil {
			return nil
		}
		return []byte(fmt.Sprintf("%s %s", cookie, encoded))
	}
}

func TestSend_RoundTrip(t *testing.T) {
	addr := newTestUDPServer(t, echoHandler(map[string]interface{}{"result": "ok"}))

	c, err := New(addr, 5*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	resp, err := c.Send(map[string]interface{}{"command": "query", "call-id": "test123"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp["result"] != "ok" {
		t.Errorf("expected result=ok, got %v", resp["result"])
	}
}

// TestSend_WireFormat verifies the request uses "<cookie> <bencode>" format and
// that cookie does not appear inside the bencode dict.
func TestSend_WireFormat(t *testing.T) {
	var received []byte
	// captured is closed by the handler after writing to received, providing the
	// happens-before guarantee needed to read received safely in the test goroutine.
	captured := make(chan struct{})
	addr := newTestUDPServer(t, func(req []byte) []byte {
		received = make([]byte, len(req))
		copy(received, req)
		close(captured)
		idx := bytes.IndexByte(req, ' ')
		if idx < 0 {
			return nil
		}
		cookie := string(req[:idx])
		encoded, _ := bencode.EncodeString(map[string]interface{}{"result": "ok"})
		return []byte(fmt.Sprintf("%s %s", cookie, encoded))
	})

	c, err := New(addr, 2*time.Second)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	cmd := map[string]interface{}{"command": "ping"}
	c.Send(cmd) //nolint:errcheck
	<-captured // wait for handler to finish writing received

	// cookie must be the prefix before the first space
	idx := bytes.IndexByte(received, ' ')
	if idx < 0 {
		t.Fatal("request has no space separator — not NG wire format")
	}
	cookiePrefix := string(received[:idx])
	if len(cookiePrefix) == 0 {
		t.Fatal("empty cookie prefix")
	}

	// bencode dict must NOT contain a "cookie" key
	var dict map[string]interface{}
	if err := bencode.DecodeString(string(received[idx+1:]), &dict); err != nil {
		t.Fatalf("bencode decode request: %v", err)
	}
	if _, ok := dict["cookie"]; ok {
		t.Error("cookie must not appear inside the bencode dict (NG protocol puts it as prefix)")
	}

	// caller's map must not be mutated
	if _, ok := cmd["cookie"]; ok {
		t.Error("Send must not mutate the caller's map")
	}

	// cookie prefix must match expected hex format (16 hex chars = 8 bytes)
	if len(cookiePrefix) != 16 || !isHex(cookiePrefix) {
		t.Errorf("unexpected cookie format: %q (want 16 hex chars)", cookiePrefix)
	}
}

func isHex(s string) bool {
	return strings.TrimLeft(s, "0123456789abcdef") == ""
}

func TestSend_Timeout(t *testing.T) {
	// Server receives but never responds.
	addr := newTestUDPServer(t, func(_ []byte) []byte { return nil })

	c, err := New(addr, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	_, err = c.Send(map[string]interface{}{"command": "query", "call-id": "x"})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout in error, got: %v", err)
	}
}
