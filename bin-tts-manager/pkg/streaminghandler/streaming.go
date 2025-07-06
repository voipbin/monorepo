package streaminghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"
)

func (h *streamingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType streaming.ReferenceType,
	referenceID uuid.UUID,
	language string,
	gender streaming.Gender,
	direction streaming.Direction,
) (*streaming.Streaming, error) {
	h.muSteaming.Lock()
	defer h.muSteaming.Unlock()

	id := h.utilHandler.UUIDCreate()
	res := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		PodID: h.podID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Language:  language,
		Gender:    gender,
		Direction: direction,

		ChanDone: make(chan bool),
	}

	_, ok := h.mapStreaming[id]
	if ok {
		return nil, fmt.Errorf("streaming already exists. streaming_id: %s", id)
	}

	h.mapStreaming[id] = res
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingCreated, res)

	return res, nil
}

// Gets returns streaming
func (h *streamingHandler) Get(ctx context.Context, streamingID uuid.UUID) (*streaming.Streaming, error) {
	h.muSteaming.Lock()
	defer h.muSteaming.Unlock()

	res, ok := h.mapStreaming[streamingID]
	if !ok {
		return nil, fmt.Errorf("streaming not found. streaming_id: %s", streamingID)
	}

	return res, nil
}

func (h *streamingHandler) Delete(ctx context.Context, streamingID uuid.UUID) {
	h.muSteaming.Lock()
	defer h.muSteaming.Unlock()

	tmp, ok := h.mapStreaming[streamingID]
	if !ok {
		return
	}

	delete(h.mapStreaming, streamingID)
	h.notifyHandler.PublishEvent(ctx, streaming.EventTypeStreamingDeleted, tmp)
}
