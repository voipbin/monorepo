package rtphandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package rtphandler -destination ./mock_rtphandler.go -source main.go -build_flags=-mod=mod

import "net"

type rtpHandler struct {
	conn    *net.UDPConn
	chanRTP chan []byte
}

// RTPHandler interface for rtp handle
type RTPHandler interface {
	Serve()
}

// NewRTPHandler returns rtphandler
func NewRTPHandler(conn *net.UDPConn, chanRTP chan []byte) RTPHandler {

	return &rtpHandler{
		conn:    conn,
		chanRTP: chanRTP,
	}
}
