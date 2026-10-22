package websockhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// i1WsPair returns a connected (server, client) websocket pair backed by an httptest server
// listening on 127.0.0.1.
func i1WsPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()

	serverCh := make(chan *websocket.Conn, 1)
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverCh <- ws
		<-release
		_ = ws.Close()
	}))

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		srv.Close()
		t.Fatalf("could not dial: %v", err)
	}

	var server *websocket.Conn
	select {
	case server = <-serverCh:
	case <-time.After(5 * time.Second):
		srv.Close()
		t.Fatalf("server side did not accept the websocket")
	}

	t.Cleanup(func() {
		_ = client.Close()
		close(release)
		srv.Close()
	})
	return server, client
}

func Test_receiveBinaryFromWebsock(t *testing.T) {
	h := &websockHandler{}

	t.Run("success", func(t *testing.T) {
		server, client := i1WsPair(t)
		if err := client.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02, 0x03}); err != nil {
			t.Fatalf("could not write: %v", err)
		}
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))

		res, err := h.receiveBinaryFromWebsock(context.Background(), server)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(res, []byte{0x01, 0x02, 0x03}) {
			t.Errorf("wrong result. got: %v", res)
		}
	})

	t.Run("wrong message type", func(t *testing.T) {
		server, client := i1WsPair(t)
		if err := client.WriteMessage(websocket.TextMessage, []byte("text")); err != nil {
			t.Fatalf("could not write: %v", err)
		}
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))

		res, err := h.receiveBinaryFromWebsock(context.Background(), server)
		if err == nil || err.Error() != "wrong message type" {
			t.Errorf("expected wrong message type error, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := h.receiveBinaryFromWebsock(ctx, nil)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
	})

	t.Run("read error on closed peer", func(t *testing.T) {
		server, client := i1WsPair(t)
		_ = client.Close()
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))

		res, err := h.receiveBinaryFromWebsock(context.Background(), server)
		if err == nil {
			t.Errorf("expected read error, got nil")
		}
		if err != nil && err.Error() == "wrong message type" {
			t.Errorf("expected a read error, got the message type error")
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
	})
}

func Test_receiveTextFromWebsock(t *testing.T) {
	h := &websockHandler{}

	t.Run("success", func(t *testing.T) {
		server, client := i1WsPair(t)
		if err := client.WriteMessage(websocket.TextMessage, []byte(`{"type":"subscribe"}`)); err != nil {
			t.Fatalf("could not write: %v", err)
		}
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))

		res, err := h.receiveTextFromWebsock(context.Background(), server)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(res) != `{"type":"subscribe"}` {
			t.Errorf("wrong result. got: %s", res)
		}
	})

	t.Run("wrong message type", func(t *testing.T) {
		server, client := i1WsPair(t)
		if err := client.WriteMessage(websocket.BinaryMessage, []byte{0x01}); err != nil {
			t.Fatalf("could not write: %v", err)
		}
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))

		res, err := h.receiveTextFromWebsock(context.Background(), server)
		if err == nil || err.Error() != "wrong message type" {
			t.Errorf("expected wrong message type error, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		res, err := h.receiveTextFromWebsock(ctx, nil)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
	})

	t.Run("read error on closed peer", func(t *testing.T) {
		server, client := i1WsPair(t)
		_ = client.Close()
		_ = server.SetReadDeadline(time.Now().Add(5 * time.Second))

		res, err := h.receiveTextFromWebsock(context.Background(), server)
		if err == nil {
			t.Errorf("expected read error, got nil")
		}
		if err != nil && err.Error() == "wrong message type" {
			t.Errorf("expected a read error, got the message type error")
		}
		if res != nil {
			t.Errorf("expected nil result, got: %v", res)
		}
	})
}
