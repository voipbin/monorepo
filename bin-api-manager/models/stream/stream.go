package stream

import (
	"net"

	"github.com/gorilla/websocket"
)

type Stream struct {
	ConnWebsocket   *websocket.Conn
	ConnAusiosocket net.Conn
}
