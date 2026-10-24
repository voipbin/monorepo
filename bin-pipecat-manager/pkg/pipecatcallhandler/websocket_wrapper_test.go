package pipecatcallhandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

// dialTestServer dials the given test server and returns the client side connection.
func dialTestServer(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("could not dial the test server: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func Test_websocketHandler_upgradeRoundTrip(t *testing.T) {
	wh := NewWebsocketHandler()
	if wh == nil {
		t.Fatalf("NewWebsocketHandler returned nil")
	}

	// the server side echoes one message through the handler wrappers.
	serverErr := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wh.Upgrade(w, r, nil)
		if err != nil {
			serverErr <- err
			return
		}
		defer func() { _ = conn.Close() }()

		msgType, data, err := wh.ReadMessage(conn)
		if err != nil {
			serverErr <- err
			return
		}
		serverErr <- wh.WriteMessage(conn, msgType, append([]byte("echo:"), data...))
	}))
	defer srv.Close()

	client := dialTestServer(t, srv)
	if err := client.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
		t.Fatalf("could not write: %v", err)
	}

	msgType, data, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("could not read: %v", err)
	}
	if msgType != websocket.TextMessage || string(data) != "echo:hello" {
		t.Errorf("wrong echo. type: %d, data: %q", msgType, data)
	}
	if errServer := <-serverErr; errServer != nil {
		t.Errorf("server side failed: %v", errServer)
	}
}

func Test_websocketHandler_upgradeRejectsPlainHTTP(t *testing.T) {
	wh := NewWebsocketHandler()

	serverErr := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := wh.Upgrade(w, r, nil)
		serverErr <- err
	}))
	defer srv.Close()

	res, err := http.Get(srv.URL) //nolint:gosec,noctx // loopback test server
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = res.Body.Close()

	if errServer := <-serverErr; errServer == nil {
		t.Errorf("expected the upgrade of a plain http request to fail")
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("wrong status. expect: %d, got: %d", http.StatusBadRequest, res.StatusCode)
	}
}

func Test_upgrader_checkOriginAllowsAnyOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://other.example.invalid")

	if !upgrader.CheckOrigin(req) {
		t.Errorf("a foreign origin must be accepted")
	}
}

func Test_websocketHandler_dialContext(t *testing.T) {
	wh := NewWebsocketHandler()

	t.Run("negotiates the media subprotocol", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			up := websocket.Upgrader{Subprotocols: []string{websocketAsteriskSubprotocol}}
			conn, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			_ = conn.Close()
		}))
		defer srv.Close()

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		conn, res, err := wh.DialContext(context.Background(), wsURL, nil) //nolint:bodyclose // gorilla/websocket replaces the handshake response body
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = conn.Close() }()

		if conn.Subprotocol() != websocketAsteriskSubprotocol {
			t.Errorf("wrong subprotocol. expect: %s, got: %s", websocketAsteriskSubprotocol, conn.Subprotocol())
		}
		if res == nil || res.StatusCode != http.StatusSwitchingProtocols {
			t.Errorf("wrong handshake response: %v", res)
		}
	})

	t.Run("fails on an unsupported scheme without a network call", func(t *testing.T) {
		conn, _, err := wh.DialContext(context.Background(), "invalid-scheme://media.test/ws", nil) //nolint:bodyclose // no response on failure
		if err == nil {
			_ = conn.Close()
			t.Fatalf("expected an error")
		}
		if conn != nil {
			t.Errorf("expected a nil connection")
		}
	})
}

func Test_websocketAsteriskConnect_errorPaths(t *testing.T) {
	newRealConn := func(t *testing.T) *websocket.Conn {
		t.Helper()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			up := websocket.Upgrader{}
			c, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer func() { _ = c.Close() }()
			for {
				if _, _, errRead := c.ReadMessage(); errRead != nil {
					return
				}
			}
		}))
		t.Cleanup(srv.Close)

		return dialTestServer(t, srv)
	}

	t.Run("set read deadline fails on a closed connection", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// ReadMessage has no expectation: it must not be reached.
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{websocketHandler: mockWS}

		conn := newRealConn(t)
		_ = conn.Close()
		mockWS.EXPECT().DialContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, &http.Response{StatusCode: http.StatusSwitchingProtocols}, nil)

		got, err := h.websocketAsteriskConnect(context.Background(), "ws://media.test/ws")
		if got != nil {
			t.Errorf("expected a nil connection")
		}
		if err == nil || !strings.Contains(err.Error(), "could not set read deadline") {
			t.Errorf("wrong error: %v", err)
		}
	})

	t.Run("read message fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{websocketHandler: mockWS}

		conn := newRealConn(t)
		mockWS.EXPECT().DialContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, &http.Response{StatusCode: http.StatusSwitchingProtocols}, nil)
		mockWS.EXPECT().ReadMessage(conn).Return(0, nil, fmt.Errorf("read failed"))

		got, err := h.websocketAsteriskConnect(context.Background(), "ws://media.test/ws")
		if got != nil {
			t.Errorf("expected a nil connection")
		}
		if err == nil || !strings.Contains(err.Error(), "could not read MEDIA_START message") || !strings.Contains(err.Error(), "read failed") {
			t.Errorf("wrong error: %v", err)
		}
	})

	t.Run("clear read deadline fails when the connection closes during the read", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{websocketHandler: mockWS}

		conn := newRealConn(t)
		mockWS.EXPECT().DialContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, &http.Response{StatusCode: http.StatusSwitchingProtocols}, nil)
		mockWS.EXPECT().ReadMessage(conn).DoAndReturn(func(c *websocket.Conn) (int, []byte, error) {
			_ = c.Close()
			return websocket.TextMessage, []byte("MEDIA_START"), nil
		})

		got, err := h.websocketAsteriskConnect(context.Background(), "ws://media.test/ws")
		if got != nil {
			t.Errorf("expected a nil connection")
		}
		if err == nil || !strings.Contains(err.Error(), "could not clear read deadline") {
			t.Errorf("wrong error: %v", err)
		}
	})
}
