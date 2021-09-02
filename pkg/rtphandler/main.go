package rtphandler

import "net"

type rtpHandler struct {
	conn    *net.UDPConn
	chanRTP chan []byte
}

// RTPHandler interface for rtp handle
type RTPHandler interface {
	Serve() error
}

// NewRTPHandler returns rtphandler
func NewRTPHandler(conn *net.UDPConn, chanRTP chan []byte) RTPHandler {

	return &rtpHandler{
		conn:    conn,
		chanRTP: chanRTP,
	}
}
