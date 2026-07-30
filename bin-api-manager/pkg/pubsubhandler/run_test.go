package pubsubhandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// newTestWebsock spins up an httptest websocket server and returns the
// client-side connection plus a channel of messages received by the server.
func newTestWebsock(t *testing.T) (*websocket.Conn, chan string, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	received := make(chan string, 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		for {
			_, m, errRead := conn.ReadMessage()
			if errRead != nil {
				return
			}
			received <- string(m)
		}
	}))

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("Could not dial the test websocket. err: %v", err)
	}

	cleanup := func() {
		_ = ws.Close()
		srv.Close()
	}

	return ws, received, cleanup
}

func Test_Run_delivers_to_websocket(t *testing.T) {
	ws, received, cleanup := newTestWebsock(t)
	defer cleanup()

	b := NewBrokerHandler()
	sub := b.NewSubHandler()
	defer sub.Terminate()

	if errSub := sub.Subscribe("topic"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chRun := make(chan error, 1)
	go func() {
		chRun <- sub.Run(ctx, ws)
	}()

	for i := 0; i < 5; i++ {
		if errPub := b.Publish("topic", fmt.Sprintf("message %d", i)); errPub != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", errPub)
		}
	}

	for i := 0; i < 5; i++ {
		select {
		case m := <-received:
			expect := fmt.Sprintf("message %d", i)
			if m != expect {
				t.Errorf("Wrong match. expect: %s, got: %s", expect, m)
			}
		case <-time.After(time.Second * 5):
			t.Errorf("Timed out waiting for message %d", i)
		}
	}

	// cancel the context and confirm the run loop exits
	cancel()
	select {
	case errRun := <-chRun:
		if errRun == nil || !strings.Contains(errRun.Error(), "context canceled") {
			t.Errorf("Wrong match. expect: context canceled, got: %v", errRun)
		}
	case <-time.After(time.Second * 5):
		t.Errorf("Timed out waiting for the run loop to exit")
	}
}

func Test_RunWithMutex_delivers_to_websocket(t *testing.T) {
	ws, received, cleanup := newTestWebsock(t)
	defer cleanup()

	b := NewBrokerHandler()
	sub := b.NewSubHandler()
	defer sub.Terminate()

	if errSub := sub.Subscribe("topic"); errSub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errSub)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var writeMu sync.Mutex
	chRun := make(chan error, 1)
	go func() {
		chRun <- sub.RunWithMutex(ctx, ws, &writeMu)
	}()

	if errPub := b.Publish("topic", "test message"); errPub != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", errPub)
	}

	select {
	case m := <-received:
		if m != "test message" {
			t.Errorf("Wrong match. expect: test message, got: %s", m)
		}
	case <-time.After(time.Second * 5):
		t.Errorf("Timed out waiting for the message")
	}

	cancel()
	select {
	case <-chRun:
	case <-time.After(time.Second * 5):
		t.Errorf("Timed out waiting for the run loop to exit")
	}
}

func Test_Run_exits_on_terminate(t *testing.T) {
	ws, _, cleanup := newTestWebsock(t)
	defer cleanup()

	b := NewBrokerHandler()
	sub := b.NewSubHandler()

	ctx := context.Background()

	chRun := make(chan error, 1)
	go func() {
		chRun <- sub.Run(ctx, ws)
	}()

	time.Sleep(time.Millisecond * 100)
	sub.Terminate()

	select {
	case errRun := <-chRun:
		if errRun != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", errRun)
		}
	case <-time.After(time.Second * 5):
		t.Errorf("Timed out waiting for the run loop to exit after Terminate")
	}
}
