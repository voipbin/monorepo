package pipecatcallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"net"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultMediaSampleRate = 16000
	defaultMediaNumChannel = 1
)

func (h *pipecatcallHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})

	log.Debugf("Listening the audiosocket stream. address: %s", h.listenAddress)
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

func (h *pipecatcallHandler) runStart(conn net.Conn) {
	log := logrus.WithField("func", "runStart")

	// Get streamingID
	streamingID, err := h.audiosocketHandler.GetStreamingID(conn)
	if err != nil {
		log.Errorf("Could not get streaming ID. err: %v", err)
		return
	}
	log = log.WithField("streaming_id", streamingID)
	log.Debugf("Found streaming id: %s", streamingID)

	// get pipecatcall info by using streamingID
	pc, err := h.Get(context.Background(), streamingID)
	if err != nil {
		log.Errorf("Could not get streaming: %v", err)
		return
	}
	log.WithField("pipecatcall", pc).Debugf("Pipecatcall info retrieved. pipecatcall_id: %s", pc.ID)

	// create a new session
	se, err := h.SessionCreate(pc, streamingID, conn)
	if err != nil {
		log.Errorf("Could not add pipecatcall session: %v", err)
		return
	}
	log.WithField("session", se).Debugf("Pipecatcall session added. pipecatcall_id: %s", pc.ID)

	// Start keep-alive in a separate goroutine
	go h.runAsteriskKeepAlive(se.Ctx, conn, defaultKeepAliveInterval, streamingID)

	go func() {
		// run the pipecat runner
		h.RunnerStart(pc, se)
		se.Cancel()
	}()

	go func() {
		// run the media handler
		h.runAsteriskReceivedMediaHandle(se)
		se.Cancel()
	}()

	<-se.Ctx.Done()

	log.Debugf("Context done, stopping pipecatcall. pipecatcall_id: %s", pc.ID)
	h.stop(context.Background(), pc)
}

func (h *pipecatcallHandler) runAsteriskKeepAlive(ctx context.Context, conn net.Conn, interval time.Duration, streamingID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runAsteriskKeepAlive",
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

func (h *pipecatcallHandler) retryWithBackoff(operation func() error, maxAttempts int, initialBackoff time.Duration) error {
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

func (h *pipecatcallHandler) runAsteriskReceivedMediaHandle(se *pipecatcall.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "runReceivedAsteriskMediaHandle",
		"pipecatcall_id": se.ID,
	})

	packetID := uint64(0)
	for {
		if se.Ctx.Err() != nil {
			log.Debugf("Context has finished. pipecatcall_id: %s", se.ID)
			return
		}

		m, err := h.audiosocketHandler.GetNextMedia(se.AsteriskConn)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		data, err := h.audiosocketHandler.Upsample8kTo16k(m.Payload())
		if err != nil {
			// invalid audio data, skip this packet
			continue
		}

		if errSend := h.pipecatframeHandler.SendAudio(se, packetID, data); errSend != nil {
			log.Errorf("Could not send audio frame. err: %v", errSend)
		}

		packetID++
	}
}
