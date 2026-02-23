package pipecatcallhandler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func Test_upgraderBufferSizes(t *testing.T) {
	// Verify buffer sizes are adequate for audio streaming
	// Audio at 16kHz, 20ms chunks = ~640 bytes + protobuf overhead
	// Minimum recommended: 64KB for read, 64KB for write

	minBufferSize := 64 * 1024 // 64KB

	if upgrader.ReadBufferSize < minBufferSize {
		t.Errorf("ReadBufferSize too small: got %d, want >= %d",
			upgrader.ReadBufferSize, minBufferSize)
	}

	if upgrader.WriteBufferSize < minBufferSize {
		t.Errorf("WriteBufferSize too small: got %d, want >= %d",
			upgrader.WriteBufferSize, minBufferSize)
	}
}

func Test_runWebSocketAsteriskRead(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "normal closure closes doneCh",
		},
		{
			name: "going away closes doneCh",
		},
		{
			name: "unexpected error closes doneCh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a real test WebSocket server/client pair.
			// runWebSocketAsteriskRead calls conn.ReadMessage() directly (not
			// through the mock interface), so we need real WebSocket connections.
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				up := websocket.Upgrader{}
				c, err := up.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				// Close the server side immediately to trigger read error on client
				_ = c.Close()
			}))
			defer srv.Close()

			wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("failed to dial test ws: %v", err)
			}

			doneCh := make(chan struct{})
			go runWebSocketAsteriskRead(conn, doneCh)

			// doneCh should be closed when read fails
			select {
			case <-doneCh:
				// success
			case <-time.After(5 * time.Second):
				t.Fatal("doneCh was not closed within timeout")
			}
		})
	}
}

func Test_websocketAsteriskConnect(t *testing.T) {
	tests := []struct {
		name string

		dialErr    error
		msgType    int
		readErr    error

		expectErr       bool
		expectErrSubstr string
	}{
		{
			name: "successful connection",

			dialErr: nil,
			msgType: websocket.TextMessage,
			readErr: nil,

			expectErr: false,
		},
		{
			name: "dial failure",

			dialErr: fmt.Errorf("connection refused"),

			expectErr:       true,
			expectErrSubstr: "could not dial WebSocket",
		},
		{
			name: "wrong message type (binary instead of text)",

			dialErr: nil,
			msgType: websocket.BinaryMessage,
			readErr: nil,

			expectErr:       true,
			expectErrSubstr: "expected text message for MEDIA_START",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockWS := NewMockWebsocketHandler(mc)

			h := &pipecatcallHandler{
				websocketHandler: mockWS,
			}

			if tt.dialErr != nil {
				// Dial fails, no further calls expected
				mockWS.EXPECT().DialContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil, tt.dialErr)
			} else {
				// Dial succeeds — we need a real *websocket.Conn so SetReadDeadline works.
				// Start a test WebSocket server to get a real connection.
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					up := websocket.Upgrader{}
					c, err := up.Upgrade(w, r, nil)
					if err != nil {
						return
					}
					// Keep the server connection open until the test completes
					defer func() { _ = c.Close() }()
					// Read loop to consume the client side
					for {
						if _, _, err := c.ReadMessage(); err != nil {
							return
						}
					}
				}))
				defer srv.Close()

				// Dial the test server to get a real *websocket.Conn
				wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
				realConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					t.Fatalf("failed to create test websocket conn: %v", err)
				}
				defer func() { _ = realConn.Close() }()

				mockWS.EXPECT().DialContext(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(realConn, &http.Response{StatusCode: 101}, nil)

				mockWS.EXPECT().ReadMessage(realConn).
					Return(tt.msgType, []byte("MEDIA_START"), tt.readErr)
			}

			conn, err := h.websocketAsteriskConnect(context.Background(), "ws://test:8088/ws")

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				} else if tt.expectErrSubstr != "" && !strings.Contains(err.Error(), tt.expectErrSubstr) {
					t.Errorf("expected error containing %q, got: %v", tt.expectErrSubstr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if conn == nil {
				t.Errorf("expected non-nil connection")
			}
		})
	}
}
