package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_websocket.go -source websocket.go -build_flags=-mod=mod

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type WebsocketHandler interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
	ReadMessage(conn *websocket.Conn) (int, []byte, error)
	WriteMessage(conn *websocket.Conn, messageType int, data []byte) error
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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
