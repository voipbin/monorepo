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

// Start starts the live streaming transcribe of the given transcribe
func (h *streamingHandler) Start(ctx context.Context, customerID uuid.UUID, transcribeID uuid.UUID, referenceType transcribe.ReferenceType, referenceID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"transcribe_id":  transcribeID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"language":       language,
		"direction":      direction,
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
	go h.runSTT(res)

	return res, nil
}

// runSTT selects the STT provider based on priority and runs the handler.
func (h *streamingHandler) runSTT(st *streaming.Streaming) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "runSTT",
		"streaming_id": st.ID,
	})

	handlers := []func(*streaming.Streaming) error{}
	for _, provider := range h.providerPriority {
		switch provider {
		case STTProviderGCP:
			handlers = append(handlers, h.gcpRun)
		case STTProviderAWS:
			handlers = append(handlers, h.awsRun)
		}
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
