package streaminghandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestWebSocketPair creates a connected client/server WebSocket pair for testing.
// Returns the client conn, a function to close the server conn, and a cleanup function.
func newTestWebSocketPair(t *testing.T) (client *websocket.Conn, closeServer func(), cleanup func()) {
	t.Helper()

	upgrader := websocket.Upgrader{}
	serverConnCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("server upgrade failed: %v", err)
		}
		serverConnCh <- conn
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("client dial failed: %v", err)
	}

	serverConn := <-serverConnCh

	return clientConn,
		func() { _ = serverConn.Close() },
		func() {
			_ = clientConn.Close()
			_ = serverConn.Close()
			srv.Close()
		}
}

func Test_runWebSocketRead_closesChannelOnDisconnect(t *testing.T) {
	client, closeServer, cleanup := newTestWebSocketPair(t)
	defer cleanup()

	doneCh := make(chan struct{})
	go runWebSocketRead(client, doneCh)

	// Close the server side to trigger a read error on the client
	closeServer()

	select {
	case <-doneCh:
		// doneCh was closed — correct behavior
	case <-time.After(2 * time.Second):
		t.Fatal("doneCh was not closed after WebSocket disconnect")
	}
}

func Test_runWebSocketRead_closesChannelOnClientClose(t *testing.T) {
	client, _, cleanup := newTestWebSocketPair(t)
	defer cleanup()

	doneCh := make(chan struct{})
	go runWebSocketRead(client, doneCh)

	// Close the client side directly (simulating Stop() closing ConnAst)
	_ = client.Close()

	select {
	case <-doneCh:
		// doneCh was closed — correct behavior
	case <-time.After(2 * time.Second):
		t.Fatal("doneCh was not closed after client close")
	}
}

func Test_websocketWrite_stopsOnContextCancel(t *testing.T) {
	client, _, cleanup := newTestWebSocketPair(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Create a large payload that needs many fragments (160 bytes each with 20ms pacing)
	// 16000 bytes = 100 fragments × 20ms = 2 seconds if not cancelled
	data := make([]byte, 16000)

	cancel() // cancel immediately

	err := websocketWrite(ctx, client, data)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func Test_websocketWrite_emptyData(t *testing.T) {
	client, _, cleanup := newTestWebSocketPair(t)
	defer cleanup()

	err := websocketWrite(context.Background(), client, nil)
	if err != nil {
		t.Fatalf("expected nil error for empty data, got: %v", err)
	}

	err = websocketWrite(context.Background(), client, []byte{})
	if err != nil {
		t.Fatalf("expected nil error for zero-length data, got: %v", err)
	}
}
