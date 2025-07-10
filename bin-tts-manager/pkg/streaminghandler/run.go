package streaminghandler

import (
	"context"
	"monorepo/bin-tts-manager/models/streaming"
	"net"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

func (h *streamingHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	log.Debugf("Listening the audiosocket stream. address: %s", h.listenAddress)
	listener, err := net.Listen("tcp", h.listenAddress)
	if err != nil {
		return errors.Wrapf(err, "could not listen on the address. address: %s", h.listenAddress)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Errorf("Error accepting connection: %v", err)
			continue
		}
		log.Debugf("Accepted connection. remote_addr: %s", conn.RemoteAddr())

		go h.runStart(conn) // Handle connection concurrently
	}
}

func (h *streamingHandler) runStart(conn net.Conn) {
	log := logrus.WithField("func", "runStart")

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
	}()

	// Get streamingID
	streamingID, err := h.audiosocketGetStreamingID(conn)
	if err != nil {
		log.Errorf("Could not get streaming ID. err: %v", err)
		return
	}
	log = log.WithField("streaming_id", streamingID)
	log.Debugf("Found streaming id: %s", streamingID)

	// // Start keep-alive in a separate goroutine
	// go h.runKeepAlive(ctx, conn, defaultKeepAliveInterval, streamingID)
	go h.runKeepConsume(ctx, conn, streamingID)

	st, err := h.Get(ctx, streamingID)
	if err != nil {
		log.Errorf("Could not get streaming: %v", err)
		return
	}
	log.WithField("streaming", st).Debugf("Streaming info retrieved. streaming_id: %s", st.ID)

	handlers := []func(context.Context, *streaming.Streaming, net.Conn) error{
		h.elevenlabsHandler.Run,
	}

	for _, handler := range handlers {
		if errRun := handler(ctx, st, conn); errRun != nil {
			log.Errorf("Handler execution failed: %v", errRun)
			continue
		}
		return
	}

	log.Warn("No handler executed successfully")
}

func (h *streamingHandler) runKeepConsume(ctx context.Context, conn net.Conn, streamingID uuid.UUID) {
	// log := logrus.WithFields(logrus.Fields{
	// 	"func":         "runKeepConsume",
	// 	"streaming_id": streamingID,
	// })

	buffer := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, err := conn.Read(buffer)
			if err != nil {
				return
			}
			// log.Debugf("Received %d bytes from connection for streaming ID: %s", n, streamingID)
		}
	}
}

func (h *streamingHandler) runKeepAlive(ctx context.Context, conn net.Conn, interval time.Duration, streamingID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runKeepAlive",
		"streaming_id": streamingID,
	})

	ticker := time.NewTicker(interval) // Use configurable interval
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("Keep-alive stopped")
			return
		case <-ticker.C:
			// Create AudioSocket keepalive message
			keepAliveMessage := []byte{0x10, 0x00, 0x00} // Header: type (0x10) + length (0x0000)

			log.Debugf("Sending keep alive message to for streaming ID: %s", streamingID)
			errRetry := h.retryWithBackoff(func() error {
				_, writeErr := conn.Write(keepAliveMessage)
				return writeErr
			}, defaultMaxRetryAttempts, defaultInitialBackoff)
			if errRetry != nil {
				log.Errorf("Failed to send keep alive message after retries: %v", errRetry)
				return
			}
		}
	}
}

func (h *streamingHandler) retryWithBackoff(operation func() error, maxAttempts int, initialBackoff time.Duration) error {
	backoff := initialBackoff
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := operation(); err != nil {
			if attempt == maxAttempts {
				return err
			}
			time.Sleep(backoff)
			backoff *= 2
		} else {
			return nil
		}
	}

	return nil
}
