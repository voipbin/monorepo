package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
)

// StreamingCreate creates a new streaming
func (h *handler) StreamingCreate(ctx context.Context, s *streaming.Streaming) error {

	// update the cache
	if errSet := h.streamingSetToCache(ctx, s); errSet != nil {
		return errSet
	}

	return nil
}

// streamingSetToCache sets the streaming to the cache.
func (h *handler) streamingSetToCache(ctx context.Context, s *streaming.Streaming) error {

	if err := h.cache.StreamingSet(ctx, s); err != nil {
		return err
	}

	return nil
}

// streamingGetFromCache gets the streaming from the cache.
func (h *handler) streamingGetFromCache(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {

	res, err := h.cache.StreamingGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// StreamingGet returns streaming.
func (h *handler) StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	return h.streamingGetFromCache(ctx, id)
}
