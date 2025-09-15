package streaminghandler

import (
	"context"
	"fmt"
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
	go h.runKeepAlive(ctx, cancel, conn, defaultKeepAliveInterval, streamingID)
	go h.runKeepConsume(ctx, cancel, conn)

	st, err := h.UpdateConnAst(streamingID, conn)
	if err != nil {
		log.Errorf("Could not update the conn ast. err: %v", err)
		return
	}

	if errStreamer := h.runStreamer(ctx, st); errStreamer != nil {
		log.Errorf("Could not run the streamer. err: %v", errStreamer)
		return
	}

	<-ctx.Done()
	log.Infof("Streaming handler stopped. streaming_id: %s", streamingID)
}

func (h *streamingHandler) runKeepConsume(ctx context.Context, cancel context.CancelFunc, conn net.Conn) {
	log := logrus.WithField("func", "runKeepConsume")

	defer cancel()

	buffer := make([]byte, 10240)
	for {
		select {
		case <-ctx.Done():
			log.Debugf("Keep-consume stopped")
			return
		default:
			_, err := conn.Read(buffer)
			if err != nil {
				log.Errorf("Error reading from connection: %v", err)
				return
			}
		}
	}
}

func (h *streamingHandler) runKeepAlive(ctx context.Context, cancel context.CancelFunc, conn net.Conn, interval time.Duration, streamingID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runKeepAlive",
		"streaming_id": streamingID,
	})
	defer cancel()

	ticker := time.NewTicker(interval) // Use configurable interval
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("Keep-alive stopped")
			return

		case <-ticker.C:
			// Create AudioSocket keepalive message
			keepAliveMessage := []byte{0x10, 0x00, 0x01, 0x00} // Header: type (0x10) + length (0x0001) + data (0x00)

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

func (h *streamingHandler) runStreamer(ctx context.Context, st *streaming.Streaming) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runStreamer",
		"streaming_id": st.ID,
	})

	initHandlers := map[streaming.VendorName]func(ctx context.Context, st *streaming.Streaming) (any, error){
		streaming.VendorNameElevenlabs: h.elevenlabsHandler.Init,
	}

	for n, f := range initHandlers {
		tmp, errInit := f(ctx, st)
		if errInit != nil {
			log.Errorf("Handler initialization failed: %v", errInit)
			continue
		}

		h.SetVendorInfo(st, n, tmp)
		break
	}

	if st.VendorConfig == nil {
		return fmt.Errorf("failed to initialize any vendor for streaming ID: %s", st.ID)
	}

	go func(s *streaming.Streaming) {

		// run the streamer based on the vendor
		switch s.VendorName {
		case streaming.VendorNameNone:
			log.Errorf("No suitable vendor found for streaming ID: %s", s.ID)
			return

		case streaming.VendorNameElevenlabs:
			log.Debugf("Starting ElevenLabs handler for streaming ID: %s", s.ID)
			if errRun := h.elevenlabsHandler.Run(s.VendorConfig); errRun != nil {
				log.Errorf("Could not run the elevenlabs handler. err: %v", errRun)
			}
			log.Debugf("ElevenLabs handler finished for streaming_id: %s, message_id: %s", s.ID, s.MessageID)

		default:
			log.Errorf("Unsupported vendor: %s for streaming ID: %s", s.VendorName, s.ID)
			return
		}

		h.SetVendorInfo(s, streaming.VendorNameNone, nil)
	}(st)

	return nil
}
