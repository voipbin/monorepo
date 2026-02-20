package streaminghandler

import (
	"bytes"
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

	// Create a large payload that needs many fragments (frameSizeUlaw bytes each with 20ms pacing)
	// 16000 bytes = 100 fragments × 20ms = 2 seconds if not cancelled
	data := make([]byte, 16000)

	cancel() // cancel immediately

	err := websocketWrite(ctx, client, data, frameSizeUlaw)
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

	err := websocketWrite(context.Background(), client, nil, frameSizeUlaw)
	if err != nil {
		t.Fatalf("expected nil error for empty data, got: %v", err)
	}

	err = websocketWrite(context.Background(), client, []byte{}, frameSizeUlaw)
	if err != nil {
		t.Fatalf("expected nil error for zero-length data, got: %v", err)
	}
}

func Test_websocketWrite_invalidFrameSize(t *testing.T) {
	client, _, cleanup := newTestWebSocketPair(t)
	defer cleanup()

	data := []byte{1, 2, 3}

	tests := []struct {
		name      string
		frameSize int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := websocketWrite(context.Background(), client, data, tt.frameSize)
			if err == nil {
				t.Fatal("expected error for invalid frameSize, got nil")
			}
		})
	}
}

// readAllFrames reads binary WebSocket frames from the server side until an error
// (connection closed). Returns concatenated data and the individual frame sizes.
func readAllFrames(t *testing.T, serverConn *websocket.Conn) ([]byte, []int) {
	t.Helper()

	var allData []byte
	var frameSizes []int

	for {
		msgType, msg, err := serverConn.ReadMessage()
		if err != nil {
			break
		}
		if msgType != websocket.BinaryMessage {
			t.Fatalf("expected binary message, got type %d", msgType)
		}
		allData = append(allData, msg...)
		frameSizes = append(frameSizes, len(msg))
	}
	return allData, frameSizes
}

func Test_websocketWrite_fragmentsData(t *testing.T) {
	tests := []struct {
		name              string
		dataLen           int
		frameSize         int
		expectedFragments int
		lastFragmentSize  int
	}{
		{
			name:              "single fragment - data smaller than frame",
			dataLen:           100,
			frameSize:         frameSizeUlaw, // 160
			expectedFragments: 1,
			lastFragmentSize:  100,
		},
		{
			name:              "single fragment - data equals frame",
			dataLen:           frameSizeUlaw, // 160
			frameSize:         frameSizeUlaw,
			expectedFragments: 1,
			lastFragmentSize:  frameSizeUlaw,
		},
		{
			name:              "multiple fragments - exact division",
			dataLen:           frameSizeUlaw * 3, // 480
			frameSize:         frameSizeUlaw,
			expectedFragments: 3,
			lastFragmentSize:  frameSizeUlaw,
		},
		{
			name:              "multiple fragments - partial last frame",
			dataLen:           frameSizeUlaw*2 + 50, // 370
			frameSize:         frameSizeUlaw,
			expectedFragments: 3,
			lastFragmentSize:  50,
		},
		{
			name:              "slin frame size",
			dataLen:           frameSizeSlin * 2, // 640
			frameSize:         frameSizeSlin,     // 320
			expectedFragments: 2,
			lastFragmentSize:  frameSizeSlin,
		},
		{
			name:              "slin16 frame size - partial",
			dataLen:           frameSizeSlin16 + 100, // 740
			frameSize:         frameSizeSlin16,        // 640
			expectedFragments: 2,
			lastFragmentSize:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build a WebSocket pair where we can read from the server side
			upgrader := websocket.Upgrader{}
			serverConnCh := make(chan *websocket.Conn, 1)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Fatalf("server upgrade failed: %v", err)
				}
				serverConnCh <- conn
			}))
			defer srv.Close()

			wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
			clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("client dial failed: %v", err)
			}
			serverConn := <-serverConnCh
			defer func() {
				_ = clientConn.Close()
				_ = serverConn.Close()
			}()

			// Fill data with a recognizable pattern
			data := make([]byte, tt.dataLen)
			for i := range data {
				data[i] = byte(i % 256)
			}

			// Write in a goroutine, read from server side
			writeDone := make(chan error, 1)
			go func() {
				writeDone <- websocketWrite(context.Background(), clientConn, data, tt.frameSize)
				_ = clientConn.Close() // close so readAllFrames stops
			}()

			received, frameSizes := readAllFrames(t, serverConn)

			if err := <-writeDone; err != nil {
				t.Fatalf("websocketWrite returned error: %v", err)
			}

			// Verify all data arrived intact
			if !bytes.Equal(received, data) {
				t.Fatalf("received data mismatch: got %d bytes, want %d bytes", len(received), len(data))
			}

			// Verify fragment count
			if len(frameSizes) != tt.expectedFragments {
				t.Fatalf("expected %d fragments, got %d (sizes: %v)", tt.expectedFragments, len(frameSizes), frameSizes)
			}

			// Verify last fragment size
			if frameSizes[len(frameSizes)-1] != tt.lastFragmentSize {
				t.Fatalf("last fragment size: got %d, want %d", frameSizes[len(frameSizes)-1], tt.lastFragmentSize)
			}

			// Verify all non-last fragments are exactly frameSize
			for i := 0; i < len(frameSizes)-1; i++ {
				if frameSizes[i] != tt.frameSize {
					t.Fatalf("fragment %d size: got %d, want %d", i, frameSizes[i], tt.frameSize)
				}
			}
		})
	}
}
