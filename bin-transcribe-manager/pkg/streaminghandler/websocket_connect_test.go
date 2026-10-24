package streaminghandler

import (
	"bufio"
	"context"
	"crypto/sha1" //nolint:gosec // the RFC 6455 opening handshake mandates SHA-1
	"encoding/base64"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

const testWebsocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// newTestMediaServer starts a loopback server that completes the websocket opening handshake
// with the "media" subprotocol and then runs the given scenario on the raw connection. It is
// written on a plain TCP listener, so this test package does not import net/http.
// The returned URL has the ws:// scheme.
func newTestMediaServer(t *testing.T, scenario func(conn net.Conn)) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		conn, errAccept := ln.Accept()
		if errAccept != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		reader := bufio.NewReader(conn)
		key := ""
		for {
			line, errRead := reader.ReadString('\n')
			if errRead != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "sec-websocket-key:") {
				key = strings.TrimSpace(line[len("sec-websocket-key:"):])
			}
		}

		sum := sha1.Sum([]byte(key + testWebsocketGUID)) //nolint:gosec // see the import
		accept := base64.StdEncoding.EncodeToString(sum[:])
		resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\nSec-WebSocket-Protocol: " + websocketSubprotocol + "\r\n\r\n"
		if _, errWrite := conn.Write([]byte(resp)); errWrite != nil {
			return
		}

		scenario(conn)
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})

	return "ws://" + ln.Addr().String()
}

// serverWriteFrame writes one unmasked server-to-client frame (payload shorter than 126 bytes).
func serverWriteFrame(conn net.Conn, opcode byte, payload []byte) {
	frame := append([]byte{0x80 | opcode, byte(len(payload))}, payload...)
	_, _ = conn.Write(frame)
}

// serverWaitClosed blocks until the client side closes the connection and returns true.
// It gives up after a bounded wait and returns false, so a missing close fails the test
// instead of hanging it.
func serverWaitClosed(conn net.Conn) bool {
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	buf := make([]byte, 512)
	for {
		if _, err := conn.Read(buf); err != nil {
			var netErr net.Error
			return !errors.As(err, &netErr) || !netErr.Timeout()
		}
	}
}

func Test_websocketConnect(t *testing.T) {
	tests := []struct {
		name string

		scenario func(conn net.Conn)

		expectErr bool
	}{
		{
			name: "text MEDIA_START message completes the handshake",
			scenario: func(conn net.Conn) {
				serverWriteFrame(conn, websocket.TextMessage, []byte("MEDIA_START channel:test"))
				// keep the connection open until the client goes away
				serverWaitClosed(conn)
			},
			expectErr: false,
		},
		{
			name: "binary first message is rejected",
			scenario: func(conn net.Conn) {
				serverWriteFrame(conn, websocket.BinaryMessage, []byte{0x01, 0x02})
				serverWaitClosed(conn)
			},
			expectErr: true,
		},
		{
			name: "server closes before sending MEDIA_START",
			scenario: func(conn net.Conn) {
				serverWriteFrame(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := newTestMediaServer(t, tt.scenario)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := websocketConnect(ctx, url)
			if tt.expectErr {
				if err == nil {
					_ = conn.Close()
					t.Fatalf("Wrong match. expect: err, got: ok")
				}
				if conn != nil {
					t.Errorf("Wrong match. expect: nil connection on failure")
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			defer func() { _ = conn.Close() }()

			if conn.Subprotocol() != websocketSubprotocol {
				t.Errorf("Wrong match. expect subprotocol: %s, got: %s", websocketSubprotocol, conn.Subprotocol())
			}
		})
	}
}

func Test_websocketConnect_dialFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// port 0 is never listening
	conn, err := websocketConnect(ctx, "ws://127.0.0.1:0/none")
	if err == nil || conn != nil {
		t.Errorf("Wrong match. expect: nil and err, got: %v and %v", conn, err)
	}
}
