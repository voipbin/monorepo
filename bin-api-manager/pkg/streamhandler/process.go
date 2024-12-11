package streamhandler

import (
	"fmt"
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *streamHandler) Process(conn net.Conn) {
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

	h.handleStreamFromAsterisk(st)
}

func (h *streamHandler) getMediaID(c net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(c)
	if err != nil {
		return uuid.Nil, err
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, fmt.Errorf("invalid message type %d getting CallID", m.Kind())
	}

	return uuid.FromBytes(m.Payload())
}
