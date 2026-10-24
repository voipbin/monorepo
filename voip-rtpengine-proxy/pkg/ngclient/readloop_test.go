package ngclient

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/zeebo/bencode"
)

// newSequenceUDPServer starts a loopback UDP server that answers every request with
// the datagrams built by respond from the request cookie, in order.
func newSequenceUDPServer(t *testing.T, respond func(cookie string) [][]byte) string {
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
			idx := bytes.IndexByte(buf[:n], ' ')
			if idx < 0 {
				continue
			}
			for _, d := range respond(string(buf[:idx])) {
				_, _ = conn.WriteTo(d, peer)
			}
		}
	}()
	return conn.LocalAddr().String()
}

func Test_readLoop_ignoresMalformedDatagrams(t *testing.T) {
	validBody, err := bencode.EncodeString(map[string]interface{}{"result": "ok"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	tests := []struct {
		name string

		// junk builds the malformed datagram sent before the valid reply
		junk func(cookie string) []byte
	}{
		{
			name: "valid bencode without a separator",
			junk: func(string) []byte { return []byte("d6:result2:oke") },
		},
		{
			name: "cookie followed by undecodable body",
			junk: func(cookie string) []byte { return []byte(cookie + " not-bencode") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := newSequenceUDPServer(t, func(cookie string) [][]byte {
				return [][]byte{tt.junk(cookie), []byte(cookie + " " + validBody)}
			})

			c, err := newNGClient(addr, 5*time.Second)
			if err != nil {
				t.Fatalf("newNGClient: %v", err)
			}
			defer c.Close()

			res, err := c.Send(map[string]interface{}{"command": "ping"})
			if err != nil {
				t.Fatalf("Wrong match. expected: ok, got: %v", err)
			}
			if res["result"] != "ok" {
				t.Errorf("Wrong match. expected: ok, got: %v", res["result"])
			}
		})
	}
}

func Test_newNGClient_resolveError(t *testing.T) {
	c, err := newNGClient("127.0.0.1:99999", time.Second)
	if err == nil {
		c.Close()
		t.Fatalf("Wrong match. expected: error, got: nil")
	}
	if !strings.Contains(err.Error(), "resolve NG address") {
		t.Errorf("Wrong match. expected resolve error, got: %v", err)
	}
}

func Test_Send_encodeError(t *testing.T) {
	addr := newTestUDPServer(t, func([]byte) []byte { return nil })
	c, err := newNGClient(addr, time.Second)
	if err != nil {
		t.Fatalf("newNGClient: %v", err)
	}
	defer c.Close()

	// a func value cannot be bencoded
	_, err = c.Send(map[string]interface{}{"command": func() {}})
	if err == nil || !strings.Contains(err.Error(), "bencode encode") {
		t.Fatalf("Wrong match. expected bencode encode error, got: %v", err)
	}

	c.mu.Lock()
	pending := len(c.pending)
	c.mu.Unlock()
	if pending != 0 {
		t.Errorf("Wrong match. expected: 0 pending, got: %d", pending)
	}
}

func Test_Send_afterClose(t *testing.T) {
	addr := newTestUDPServer(t, func([]byte) []byte { return nil })
	c, err := newNGClient(addr, time.Second)
	if err != nil {
		t.Fatalf("newNGClient: %v", err)
	}
	c.Close()

	_, err = c.Send(map[string]interface{}{"command": "ping"})
	if err == nil || !strings.Contains(err.Error(), "send NG command") {
		t.Fatalf("Wrong match. expected send error, got: %v", err)
	}

	c.mu.Lock()
	pending := len(c.pending)
	c.mu.Unlock()
	if pending != 0 {
		t.Errorf("Wrong match. expected: 0 pending, got: %d", pending)
	}

	// closing an already closed client only logs a warning
	c.Close()
}
