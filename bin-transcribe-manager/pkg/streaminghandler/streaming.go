package streaminghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcript"
)

func (h *streamingHandler) Create(ctx context.Context, customerID uuid.UUID, treanscribeID uuid.UUID, externalMediaID uuid.UUID, language string, direction transcript.Direction) (*streaming.Streaming, error) {
	// log := logrus.WithFields(logrus.Fields{
	// 	"func": "Create",
	// })
	h.muSteaming.Lock()
	defer h.muSteaming.Unlock()

	id := h.utilHandler.UUIDCreate()
	res := &streaming.Streaming{
		ID:              id,
		CustomerID:      customerID,
		TranscribeID:    treanscribeID,
		ExternalMediaID: externalMediaID,
		Language:        language,
		Direction:       direction,
	}

	h.mapStreaming[id] = res
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingStarted, res)

	return res, nil

	// if errCreate := h.db.StreamingCreate(ctx, tmp); errCreate != nil {
	// 	log.Errorf("Could not create streaming. err: %v", errCreate)
	// 	return nil, errCreate
	// }

	// res, err := h.db.StreamingGet(ctx, id)
	// if err != nil {
	// 	log.Errorf("Could not get created streaming. err: %v", err)
	// 	return nil, err
	// }
	// h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingStarted, res)

	// return res, nil
}

// Gets returns streaming
func (h *streamingHandler) Get(ctx context.Context, streamingID uuid.UUID) (*streaming.Streaming, error) {
	// log := logrus.WithFields(logrus.Fields{
	// 	"func":         "Get",
	// 	"streaming_id": streamingID,
	// })

	h.muSteaming.Lock()
	defer h.muSteaming.Unlock()

	res, ok := h.mapStreaming[streamingID]
	if !ok {
		return nil, fmt.Errorf("streaming not found. streaming_id: %s", streamingID)
	}

	return res, nil

	// res, err := h.db.StreamingGet(ctx, streamingID)
	// if err != nil {
	// 	log.Errorf("Could not get streaming. err: %v", err)
	// 	return nil, err
	// }

	// return res, nil
}

func (h *streamingHandler) Delete(ctx context.Context, streamingID uuid.UUID) {
	h.muSteaming.Lock()
	defer h.muSteaming.Unlock()

	tmp, ok := h.mapStreaming[streamingID]
	if !ok {
		return
	}

	delete(h.mapStreaming, streamingID)
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingStopped, tmp)
}
