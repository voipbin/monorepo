package streaminghandler

import (
	"fmt"
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
)

// audiosocketGetStreamingID gets the streaming id from the connection
// the first message of the audiosocket should be the streaming id
func (h *streamingHandler) audiosocketGetStreamingID(conn net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(conn)
	if err != nil {
		return uuid.Nil, err
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, fmt.Errorf("wrong message kind. kind: %v", m.Kind())
	}

	res := uuid.FromBytesOrNil(m.Payload())
	return res, nil
}

func (h *streamingHandler) audiosocketGetNextMedia(conn net.Conn) (audiosocket.Message, error) {
	m, err := audiosocket.NextMessage(conn)
	if err != nil {
		return nil, err
	}

	if m.Kind() != audiosocket.KindSlin {
		if m.Kind() == audiosocket.KindHangup {
			return nil, fmt.Errorf("received hangup")
		}

		return nil, nil
	}

	if m.ContentLength() < 1 {
		return nil, nil
	}

	return m, nil
}
