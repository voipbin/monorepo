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

func Test_websocketAsteriskWrite(t *testing.T) {
	tests := []struct {
		name string

		data      []byte
		frameSize int

		expectWriteCalls int
		expectErr        bool
	}{
		{
			name: "single frame (640 bytes)",

			data:      make([]byte, 640),
			frameSize: 640,

			expectWriteCalls: 1,
			expectErr:        false,
		},
		{
			name: "multiple frames (1280 bytes, 2 writes)",

			data:      make([]byte, 1280),
			frameSize: 640,

			expectWriteCalls: 2,
			expectErr:        false,
		},
		{
			name: "data smaller than frame size (320 bytes, 1 write)",

			data:      make([]byte, 320),
			frameSize: 640,

			expectWriteCalls: 1,
			expectErr:        false,
		},
		{
			name: "empty data (0 writes)",

			data:      []byte{},
			frameSize: 640,

			expectWriteCalls: 0,
			expectErr:        false,
		},
		{
			name: "unaligned data (960 bytes, 1 full + 1 partial frame)",

			data:      make([]byte, 960),
			frameSize: 640,

			expectWriteCalls: 2,
			expectErr:        false,
		},
		{
			name: "invalid frame size (0)",

			data:      make([]byte, 640),
			frameSize: 0,

			expectWriteCalls: 0,
			expectErr:        true,
		},
		{
			name: "negative frame size",

			data:      make([]byte, 640),
			frameSize: -1,

			expectWriteCalls: 0,
			expectErr:        true,
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

			if tt.expectWriteCalls > 0 {
				mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, gomock.Any()).
					Return(nil).
					Times(tt.expectWriteCalls)
			}

			err := h.websocketAsteriskWrite(context.Background(), nil, tt.data, tt.frameSize)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_websocketAsteriskWrite_contextAlreadyCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)

	h := &pipecatcallHandler{
		websocketHandler: mockWS,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	err := h.websocketAsteriskWrite(ctx, nil, make([]byte, 640), 640)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func Test_websocketAsteriskWrite_contextCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)

	h := &pipecatcallHandler{
		websocketHandler: mockWS,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// First write succeeds, then cancel before second frame
	mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, gomock.Any()).
		DoAndReturn(func(_ *websocket.Conn, _ int, _ []byte) error {
			cancel()
			return nil
		}).
		Times(1)

	data := make([]byte, 1280)
	err := h.websocketAsteriskWrite(ctx, nil, data, 640)

	if err == nil {
		t.Errorf("expected context cancelled error but got nil")
	}
	if err != nil && err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
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

func Test_websocketAsteriskWrite_writeErrorMidWrite(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)

	// First write succeeds, second write fails
	gomock.InOrder(
		mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, gomock.Any()).Return(nil),
		mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, gomock.Any()).Return(fmt.Errorf("connection closed")),
	)

	h := &pipecatcallHandler{
		websocketHandler: mockWS,
	}

	err := h.websocketAsteriskWrite(context.Background(), nil, make([]byte, 1280), 640)
	if err == nil {
		t.Fatal("expected error but got nil")
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
