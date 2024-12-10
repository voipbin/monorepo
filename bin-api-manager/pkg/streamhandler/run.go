package streamhandler

import (
	"net"

	"github.com/sirupsen/logrus"
)

func (h *streamHandler) Run(conn net.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	mediaID, err := h.getMediaID(conn)
	if err != nil {
		log.Errorf("Could not get media ID. err: %v", err)
		conn.Close()
		return
	}

	st, err := h.SetAudiosock(mediaID, conn)
	if err != nil {
		log.Errorf("Could not set audiosock. err: %v", err)
		conn.Close()
		return
	}
	defer h.Terminate(mediaID)

	h.handleStreamFromAudiosocket(st)
}
