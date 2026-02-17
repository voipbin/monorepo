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
		"provider":     st.Provider,
	})

	// Select handler based on provider
	handler, vendorName := h.getStreamerByProvider(st.Provider)
	if handler == nil {
		return fmt.Errorf("unsupported or unconfigured provider: %s", st.Provider)
	}

	tmp, errInit := handler.Init(ctx, st)
	if errInit != nil {
		log.Errorf("Handler initialization failed for provider %s: %v", st.Provider, errInit)
		return fmt.Errorf("could not initialize %s handler: %v", st.Provider, errInit)
	}

	h.SetVendorInfo(st, vendorName, tmp)

	go func(s *streaming.Streaming) {
		log.Debugf("Starting %s handler for streaming ID: %s", s.VendorName, s.ID)
		if errRun := handler.Run(s.VendorConfig); errRun != nil {
			log.Errorf("Could not run the %s handler. err: %v", s.VendorName, errRun)
		}
		log.Debugf("%s handler finished for streaming_id: %s, message_id: %s", s.VendorName, s.ID, s.MessageID)

		h.SetVendorInfo(s, streaming.VendorNameNone, nil)
	}(st)

	return nil
}

// getStreamerByProvider returns the streamer handler and vendor name for the given provider string.
func (h *streamingHandler) getStreamerByProvider(provider string) (streamer, streaming.VendorName) {
	switch streaming.VendorName(provider) {
	case streaming.VendorNameElevenlabs:
		return h.elevenlabsHandler, streaming.VendorNameElevenlabs
	case streaming.VendorNameGCP:
		return h.gcpHandler, streaming.VendorNameGCP
	case streaming.VendorNameAWS:
		return h.awsHandler, streaming.VendorNameAWS
	default:
		// Fallback: try elevenlabs for empty/unknown provider (backwards compat)
		return h.elevenlabsHandler, streaming.VendorNameElevenlabs
	}
}
