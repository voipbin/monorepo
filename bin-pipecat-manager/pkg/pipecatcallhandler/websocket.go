package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_websocket.go -source websocket.go -build_flags=-mod=mod

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	websocketAsteriskSubprotocol = "media"
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
