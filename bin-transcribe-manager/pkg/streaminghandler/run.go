package streaminghandler

import (
	"context"
	"fmt"
	"monorepo/bin-transcribe-manager/models/streaming"
	"net"

	"github.com/pkg/errors"

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
	streamingID, err := h.audiosocketGetStreamingID(conn)
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

	handlers := []func(*streaming.Streaming, net.Conn) error{
		h.gcpRun,
		h.awsRun,
	}

	for _, handler := range handlers {
		if errRun := handler(st, conn); errRun == nil {
			return
		} else {
			log.Errorf("Could not run the handler. err: %v", errRun)
		}
	}
}
