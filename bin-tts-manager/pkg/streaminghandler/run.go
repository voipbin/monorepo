package streaminghandler

import (
	"context"
	"fmt"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/sirupsen/logrus"
)

// Run is a no-op. With WebSocket transport, tts-manager dials out to Asterisk
// per-session instead of listening on a TCP port. Kept to satisfy the
// StreamingHandler interface.
func (h *streamingHandler) Run() error {
	return nil
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
