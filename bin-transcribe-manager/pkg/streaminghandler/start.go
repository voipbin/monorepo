package streaminghandler

import (
	"context"

	"monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// defaultProviderOrder is the hard-coded fallback order.
var defaultProviderOrder = []STTProvider{STTProviderGCP, STTProviderAWS}

// getProviderFunc returns the handler function for the given provider,
// or nil if the provider is not initialized.
func (h *streamingHandler) getProviderFunc(p STTProvider) func(*streaming.Streaming) error {
	switch p {
	case STTProviderGCP:
		if h.gcpClient != nil {
			return h.gcpRun
		}
	case STTProviderAWS:
		if h.awsClient != nil {
			return h.awsRun
		}
	}
	return nil
}

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction, provider transcribe.Provider) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"transcribe_id":  transcribeID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
		"provider":       provider,
	})

	// create streaming record
	res, err := h.Create(ctx, customerID, transcribeID, language, direction)
	if err != nil {
		log.Errorf("Could not create streaming. err: %v", err)
		return nil, err
	}
	log.WithField("streaming", res).Debugf("Created a new streaming. streaming_id: %s", res.ID)

	// start the external media via call-manager
	em, err := h.reqHandler.CallV1ExternalMediaStart(
		ctx,
		res.ID,
		externalmedia.ReferenceType(referenceType),
		referenceID,
		"INCOMING",
		defaultEncapsulation,
		defaultTransport,
		"", // transportData
		defaultConnectionType,
		defaultFormat,
		externalmedia.Direction(direction),
		externalmedia.DirectionNone,
	)
	if err != nil {
		log.Errorf("Could not create external media. err: %v", err)
		return nil, err
	}
	log.WithField("external_media", em).Debugf("Started external media. external_media_id: %s, media_uri: %s", em.ID, em.MediaURI)

	// connect to Asterisk via WebSocket
	conn, err := websocketConnect(ctx, em.MediaURI)
	if err != nil {
		log.Errorf("Could not connect WebSocket to Asterisk. err: %v", err)
		// clean up the orphaned external media channel and streaming record
		if _, errStop := h.reqHandler.CallV1ExternalMediaStop(ctx, em.ID); errStop != nil {
			log.Errorf("Could not stop orphaned external media. err: %v", errStop)
		}
		h.Delete(ctx, res.ID)
		return nil, err
	}
	log.Debugf("WebSocket connected to Asterisk. media_uri: %s", em.MediaURI)

	// store connection on streaming record
	res.ConnAst = conn

	// spawn STT processing — the media processor's ReadMessage loop handles
	// ping/pong/close frames automatically, so no separate read goroutine needed.
	go h.runSTT(res, provider)

	return res, nil
}

// runSTT selects the STT provider and runs the handler.
// If provider is set, tries that provider first then falls back to default order.
// If provider is empty, uses the default order (GCP -> AWS).
func (h *streamingHandler) runSTT(st *streaming.Streaming, provider transcribe.Provider) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runSTT",
		"streaming_id": st.ID,
		"provider":     provider,
	})

	handlers := []func(*streaming.Streaming) error{}

	// If customer requested a specific provider, try it first
	if provider != "" {
		sttProvider, err := validateProvider(string(provider))
		if err == nil {
			if fn := h.getProviderFunc(sttProvider); fn != nil {
				handlers = append(handlers, fn)
			} else {
				log.Warnf("Requested provider %q not initialized, falling back to default order", provider)
			}
		} else {
			log.Warnf("Invalid provider %q, falling back to default order", provider)
		}
	}

	// Append default order (skip any already added as first choice)
	for _, p := range defaultProviderOrder {
		fn := h.getProviderFunc(p)
		if fn == nil {
			continue
		}
		if len(handlers) > 0 && provider != "" {
			requested, _ := validateProvider(string(provider))
			if p == requested {
				continue
			}
		}
		handlers = append(handlers, fn)
	}

	if len(handlers) == 0 {
		log.Error("No STT providers available for transcription")
		return
	}

	for _, handler := range handlers {
		if errRun := handler(st); errRun != nil {
			log.Errorf("Handler execution failed: %v", errRun)
			continue
		}
		return
	}

	log.Warn("No handler executed successfully")
}
