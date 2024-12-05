package streamhandler

import (
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func (h *streamHandler) ProcessStreamsock(conn net.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func": "ProcessStreamsock",
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

	for {

		m, errRecv := audiosocket.NextMessage(st.ConnAusiosocket)
		if errRecv != nil {
			log.Errorf("Could not receive audiosock data. err: %v", errRecv)
			return
		}

		switch {
		case m.Kind() == audiosocket.KindError:
			log.Debugf("Received error. err: %d", m.ErrorCode())
			continue

		case m.Kind() != audiosocket.KindSlin:
			log.Debugf("Ignoring non-slin message")
			continue

		case m.ContentLength() < 1:
			log.Debugf("No content")
			continue

		default:
			if errWrite := st.ConnWebsocket.WriteMessage(websocket.BinaryMessage, m); errWrite != nil {
				log.Debugf("Could not write the message. err: %v", errWrite)
				return
			}
		}
	}
}
