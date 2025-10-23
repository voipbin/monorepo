package pipecatcallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		_ = conn.Close()
	}()

	// Get streamingID
	streamingID, err := h.audiosocketHandler.GetStreamingID(conn)
	if err != nil {
		log.Errorf("Could not get streaming ID. err: %v", err)
		return
	}
	log = log.WithField("streaming_id", streamingID)
	log.Debugf("Found streaming id: %s", streamingID)

	// Start keep-alive in a separate goroutine
	go h.runKeepAlive(ctx, conn, defaultKeepAliveInterval, streamingID)

	// get pipecatcall info by using streamingID
	pc, err := h.Get(ctx, streamingID)
	if err != nil {
		log.Errorf("Could not get streaming: %v", err)
		return
	}
	h.setAsteriskInfo(pc, streamingID, conn)
	log.WithField("streaming", pc).Debugf("Streaming info retrieved. streaming_id: %s", pc.ID)

	go func() {
		// run the pipecat runner
		h.RunnerStart(ctx, pc)
		cancel()
	}()

	go func() {
		// run the media handler
		h.mediaStart(ctx, pc)
		cancel()
	}()

	<-ctx.Done()

	log.Debugf("Context done, stopping pipecatcall. pipecatcall_id: %s", pc.ID)
	h.stop(context.Background(), pc)
}

func (h *pipecatcallHandler) runKeepAlive(ctx context.Context, conn net.Conn, interval time.Duration, streamingID uuid.UUID) {
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

func (h *pipecatcallHandler) mediaStart(ctx context.Context, pc *pipecatcall.Pipecatcall) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "mediaRun",
		"pipecatcall": pc.ID,
	})

	packetID := uint64(0)
	for {
		if ctx.Err() != nil {
			log.Debugf("Context has finished. pipecatcall_id: %s", pc.ID)
			return
		}

		m, err := h.audiosocketHandler.GetNextMedia(pc.AsteriskConn)
		if err != nil {
			log.Infof("Connection has closed. err: %v", err)
			return
		}

		data, err := h.audiosocketHandler.Upsample8kTo16k(m.Payload())
		if err != nil {
			// invalid audio data, skip this packet
			continue
		}

		pipecatFrame := &pipecatframe.Frame{
			Frame: &pipecatframe.Frame_Audio{
				Audio: &pipecatframe.AudioRawFrame{
					Id:          packetID,
					Audio:       data,
					SampleRate:  defaultMediaSampleRate,
					NumChannels: defaultMediaNumChannel,
				},
			},
		}

		if pc.RunnerWebsocket != nil {
			if errSend := h.sendProtobufFrame(pc.RunnerWebsocket, pipecatFrame); errSend != nil {
				log.Errorf("Could not send the frame. err: %v", errSend)
			}
		}
		packetID++
	}
}
