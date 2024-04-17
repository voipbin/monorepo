package streaminghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

func (h *streamingHandler) Create(ctx context.Context, customerID uuid.UUID, treanscribeID uuid.UUID, externalMediaID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
	})

	id := h.utilHandler.UUIDCreate()
	tmp := &streaming.Streaming{
		ID:              id,
		CustomerID:      customerID,
		TranscribeID:    treanscribeID,
		ExternalMediaID: externalMediaID,
		Language:        language,
		Direction:       direction,
	}

	if errCreate := h.db.StreamingCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create streaming. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.db.StreamingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created streaming. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingStarted, res)

	return res, nil
}

// Gets returns streaming
func (h *streamingHandler) Get(ctx context.Context, streamingID uuid.UUID) (*streaming.Streaming, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "Get",
			"streaming_id": streamingID,
		},
	)

	res, err := h.db.StreamingGet(ctx, streamingID)
	if err != nil {
		log.Errorf("Could not get streaming. err: %v", err)
		return nil, err
	}

	return res, nil
}
