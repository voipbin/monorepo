package eventhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
)

var upgrader = websocket.Upgrader{} // use default options

// newARIServer starts a loopback ARI websocket server on its own mux. The
// handler is called for every websocket connection on /ari/events.
func newARIServer(t *testing.T, handler func(c *websocket.Conn, r *http.Request)) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/ari/events", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() {
			_ = c.Close()
		}()

		handler(c, r)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts
}

func hostOf(t *testing.T, ts *httptest.Server) string {
	t.Helper()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("could not parse the server url. err: %v", err)
	}

	return u.Host
}

func echo(c *websocket.Conn, r *http.Request) {
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			break
		}
		if err := c.WriteMessage(mt, message); err != nil {
			break
		}
	}
}

func Test_eventARIRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ts := newARIServer(t, echo)

	h := eventHandler{
		ariAddr: hostOf(t, ts),
	}

	if err := h.eventARIConnect(); err != nil {
		t.Fatalf("Expected ok, but got an error. err: %v", err)
	}
	defer func() {
		_ = h.ariSock.Close()
	}()
}

func Test_eventARIConnect_query(t *testing.T) {
	queries := make(chan url.Values, 1)
	ts := newARIServer(t, func(c *websocket.Conn, r *http.Request) {
		queries <- r.URL.Query()
	})

	h := eventHandler{
		ariAddr:         hostOf(t, ts),
		ariAccount:      "fake-user:fake-pass",
		ariSubscribeAll: "true",
		ariApplication:  "test-app",
	}

	if err := h.eventARIConnect(); err != nil {
		t.Fatalf("Expected ok, but got an error. err: %v", err)
	}
	defer func() {
		_ = h.ariSock.Close()
	}()

	select {
	case q := <-queries:
		if q.Get("api_key") != "fake-user:fake-pass" || q.Get("subscribeAll") != "true" || q.Get("app") != "test-app" {
			t.Errorf("Wrong query. api_key: %q, subscribeAll: %q, app: %q", q.Get("api_key"), q.Get("subscribeAll"), q.Get("app"))
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the server did not see the connection in time")
	}
}

func Test_eventARIConnect_unreachable(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	addr := hostOf(t, ts)
	ts.Close()

	h := eventHandler{ariAddr: addr}
	if err := h.eventARIConnect(); err == nil {
		_ = h.ariSock.Close()
		t.Errorf("Expected an error, but got ok")
	}
}

func Test_eventARIReceive(t *testing.T) {
	tests := []struct {
		name string

		message string
		// closeFirst makes the server close the socket without sending anything.
		closeFirst bool

		expectErr string
	}{
		{
			name: "publishes the raw message",

			message: `{"type":"StasisStart","application":"test-app"}`,
		},
		{
			name: "read failure is wrapped and nothing is published",

			closeFirst: true,

			expectErr: "Could not read message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			ts := newARIServer(t, func(c *websocket.Conn, r *http.Request) {
				if tt.closeFirst {
					return
				}
				_ = c.WriteMessage(websocket.TextMessage, []byte(tt.message))
				// keep the socket open until the client goes away.
				for {
					if _, _, err := c.ReadMessage(); err != nil {
						return
					}
				}
			})

			h := eventHandler{
				notifyhandler: mockNotify,
				ariAddr:       hostOf(t, ts),
			}
			if err := h.eventARIConnect(); err != nil {
				t.Fatalf("Expected ok, but got an error. err: %v", err)
			}
			defer func() {
				_ = h.ariSock.Close()
			}()

			ctx := context.Background()
			if tt.expectErr == "" {
				mockNotify.EXPECT().PublishEventRaw(ctx, "ari_event", "application/json", []byte(tt.message))
			}

			err := h.eventARIReceive(ctx)
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_Run_reconnect_loop_publishes_events(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	// only the first connection receives a message and is then closed. every
	// later connection (the loop reconnects after one second) is dropped
	// without a message so the mock sees exactly one publication.
	var conns atomic.Int32
	ts := newARIServer(t, func(c *websocket.Conn, r *http.Request) {
		if conns.Add(1) != 1 {
			return
		}
		_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"ChannelCreated"}`))
	})

	published := make(chan struct{}, 1)
	mockNotify.EXPECT().PublishEventRaw(gomock.Any(), "ari_event", "application/json", []byte(`{"type":"ChannelCreated"}`)).
		Do(func(ctx context.Context, eventType string, dataType string, data []byte) {
			published <- struct{}{}
		}).Times(1)

	h := &eventHandler{
		notifyhandler: mockNotify,
		ariAddr:       hostOf(t, ts),
	}
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-published:
	case <-time.After(5 * time.Second):
		t.Fatalf("the event was not published in time")
	}

	// the loop reconnects after the server closed the first socket.
	deadline := time.Now().Add(5 * time.Second)
	for conns.Load() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("the loop did not reconnect after the socket closed")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
