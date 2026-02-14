package streaminghandler

import (
	"context"
	"fmt"
	"monorepo/bin-tts-manager/models/streaming"
	"net"
	"time"

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
		_ = conn.Close()
	}()

	// Get streamingID
	streamingID, err := h.audiosocketGetStreamingID(conn)
	if err != nil {
		log.Errorf("Could not get streaming ID. err: %v", err)
		return
	}
	log = log.WithField("streaming_id", streamingID)
	log.Debugf("Found streaming id: %s", streamingID)

	// Start silence feed to prevent Asterisk from tearing down the AudioSocket channel
	go h.runSilenceFeed(ctx, cancel, conn)
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

// runSilenceFeed sends 20ms silence frames to the Asterisk AudioSocket connection
// at regular intervals. This prevents Asterisk's audiosocket_read from getting EAGAIN
// (Resource temporarily unavailable) and tearing down the channel.
//
// Asterisk's bridge media loop reads audio frames every ~20ms. If no data is available,
// res_audiosocket.c returns an error and chan_audiosocket.c hangs up the channel.
// This function keeps the connection alive by sending silence (zero-filled PCM frames).
func (h *streamingHandler) runSilenceFeed(ctx context.Context, cancel context.CancelFunc, conn net.Conn) {
	log := logrus.WithField("func", "runSilenceFeed")
	defer cancel()

	silenceData := make([]byte, audiosocketSilenceFrameSize)
	silenceFrame, err := audiosocketWrapDataPCM16Bit(silenceData)
	if err != nil {
		log.Errorf("Failed to create silence frame: %v", err)
		return
	}

	ticker := time.NewTicker(defaultSilenceFeedInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("Silence feed stopped")
			return

		case <-ticker.C:
			if _, errWrite := conn.Write(silenceFrame); errWrite != nil {
				log.Errorf("Failed to send silence frame: %v", errWrite)
				return
			}
		}
	}
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
