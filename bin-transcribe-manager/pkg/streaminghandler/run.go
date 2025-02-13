package streaminghandler

import (
	"context"
	"fmt"
	"net"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/sirupsen/logrus"
)

func (h *streamingHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	log.Debugf("Listening the audiosocket steram. address: %s", h.listenAddress)
	listener, err := net.Listen("tcp", h.listenAddress)
	if err != nil {
		return errors.Wrapf(err, "could not listen on the address. addres: %s", h.listenAddress)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		log.Debugf("Accepted connection. remote_addr: %s", conn.RemoteAddr())

		go h.runStart(conn) // Handle connection concurrently
	}
}

func (h *streamingHandler) runStart(conn net.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func": "runStart",
	})

	// get streamingID
	streamingID, err := h.getStreamingID(conn)
	if err != nil {
		log.Errorf("Could not get mediaID. err: %v", err)
		return
	}
	log.Debugf("Found transcribe id. transcribe_id: %s", streamingID)

	st, err := h.Get(context.Background(), streamingID)
	if err != nil {
		log.Errorf("Could not get streaming. err: %v", err)
		return
	}

	if errStart := h.gcpStartTCP(st, conn); errStart != nil {
		log.Errorf("Could not start the gcp. err: %v", errStart)
	}
}

func (h *streamingHandler) getStreamingID(c net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(c)
	if err != nil {
		return uuid.Nil, err
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, fmt.Errorf("wrong message kind: %v", m.Kind())
	}

	res := uuid.FromBytesOrNil(m.Payload())
	return res, nil
}
