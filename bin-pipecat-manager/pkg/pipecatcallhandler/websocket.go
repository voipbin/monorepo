package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_websocket.go -source websocket.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	websocketAsteriskSubprotocol = "media"
	websocketAsteriskWriteDelay  = 20 * time.Millisecond
	websocketAsteriskFrameSize   = 640 // 16000 Hz * 2 bytes * 20ms
)

type WebsocketHandler interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
	ReadMessage(conn *websocket.Conn) (int, []byte, error)
	WriteMessage(conn *websocket.Conn, messageType int, data []byte) error
	DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024, // 64KB - adequate for audio frames + protobuf
	WriteBufferSize: 64 * 1024, // 64KB - adequate for audio frames + protobuf
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type websocketHandler struct{}

func NewWebsocketHandler() WebsocketHandler {
	return &websocketHandler{}
}

func (h *websocketHandler) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, responseHeader)
}

func (h *websocketHandler) ReadMessage(conn *websocket.Conn) (int, []byte, error) {
	return conn.ReadMessage()
}

func (h *websocketHandler) WriteMessage(conn *websocket.Conn, messageType int, data []byte) error {
	return conn.WriteMessage(messageType, data)
}

func (h *websocketHandler) DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	dialer := websocket.Dialer{
		Subprotocols: []string{websocketAsteriskSubprotocol},
	}
	return dialer.DialContext(ctx, urlStr, requestHeader)
}

// websocketAsteriskConnect dials the Asterisk chan_websocket endpoint and waits
// for the MEDIA_START text message that signals the channel is ready.
func (h *pipecatcallHandler) websocketAsteriskConnect(ctx context.Context, mediaURI string) (*websocket.Conn, error) {
	conn, _, err := h.websocketHandler.DialContext(ctx, mediaURI, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "could not dial WebSocket. media_uri: %s", mediaURI)
	}

	if errDeadline := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); errDeadline != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(errDeadline, "could not set read deadline")
	}

	msgType, _, err := h.websocketHandler.ReadMessage(conn)
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(err, "could not read MEDIA_START message")
	}

	if errDeadline := conn.SetReadDeadline(time.Time{}); errDeadline != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(errDeadline, "could not clear read deadline")
	}

	if msgType != websocket.TextMessage {
		_ = conn.Close()
		return nil, errors.Errorf("expected text message for MEDIA_START, got type %d", msgType)
	}

	return conn, nil
}

// websocketAsteriskWrite fragments and sends raw audio data over a WebSocket
// connection as binary frames with 20ms pacing. frameSize is the number of
// bytes per 20ms frame for the channel's audio format.
func (h *pipecatcallHandler) websocketAsteriskWrite(ctx context.Context, conn *websocket.Conn, data []byte, frameSize int) error {
	if len(data) == 0 {
		return nil
	}
	if frameSize <= 0 {
		return fmt.Errorf("frameSize must be positive, got %d", frameSize)
	}

	ticker := time.NewTicker(websocketAsteriskWriteDelay)
	defer ticker.Stop()

	offset := 0
	payloadLen := len(data)

	for offset < payloadLen {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fragmentLen := min(frameSize, payloadLen-offset)
		fragment := data[offset : offset+fragmentLen]

		if err := h.websocketHandler.WriteMessage(conn, websocket.BinaryMessage, fragment); err != nil {
			return errors.Wrapf(err, "failed to write WebSocket binary frame")
		}

		offset += fragmentLen

		if offset >= payloadLen {
			break
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// runWebSocketAsteriskRead reads from the WebSocket connection to handle
// ping/pong and close frames. Without a read loop, gorilla/websocket won't
// acknowledge pings. Closes doneCh when the connection is closed or encounters
// an error, signalling handlers to tear down their sessions.
func runWebSocketAsteriskRead(conn *websocket.Conn, doneCh chan struct{}) {
	log := logrus.WithField("func", "runWebSocketAsteriskRead")
	defer close(doneCh)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Debugf("Asterisk WebSocket closed normally: %v", err)
			} else {
				log.Errorf("Asterisk WebSocket read error: %v", err)
			}
			return
		}
	}
}
