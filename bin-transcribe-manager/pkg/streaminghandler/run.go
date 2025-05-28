package streaminghandler

import (
	"context"
	"fmt"
	"monorepo/bin-transcribe-manager/models/streaming"
	"net"
	"time"

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
	defer conn.Close()

	// get streamingID
	streamingID, err := h.audiosocketGetStreamingID(conn)
	if err != nil {
		log.Errorf("Could not get mediaID. err: %v", err)
		return
	}
	log.Debugf("Found streaming id. streaming_id: %s", streamingID)

	st, err := h.Get(context.Background(), streamingID)
	if err != nil {
		log.Errorf("Could not get streaming. err: %v", err)
		return
	}
	log.WithField("streaming", st).Debugf("Found streaming info. streaming_id: %s", st.ID)

	handlers := []func(*streaming.Streaming, net.Conn) error{
		h.gcpRun,
		h.awsRun,
	}

	// Start Keep Alive after successful handler execution
	go h.startKeepAlive(conn)

	for _, handler := range handlers {
		if errRun := handler(st, conn); errRun == nil {
			return
		} else {
			log.Errorf("Could not run the handler. err: %v", errRun)
		}
	}
}

func (h *streamingHandler) startKeepAlive(conn net.Conn) {
	log := logrus.WithFields(logrus.Fields{
		"func": "startKeepAlive",
	})

	ticker := time.NewTicker(10 * time.Second) // Send keep alive every 10 seconds
	defer ticker.Stop()

	// Use for range to iterate over ticker.C
	for range ticker.C {
		// Create AudioSocket keepalive message
		// Header: type (0x10) + length (0x0000)
		keepAliveMessage := []byte{0x10, 0x00, 0x00}

		// Send message
		_, err := conn.Write(keepAliveMessage)
		if err != nil {
			log.Debugf("Failed to send keep alive message. err: %v", err)
			return
		}
	}
}
