package dbhandler

import (
	"context"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// StreamingCreate creates a new variable.
func (h *dbHandler) StreamingCreate(ctx context.Context, t *streaming.Streaming) error {
	return h.streamingSetToCache(ctx, t)
}

// streamingSetToCache sets the given streaming to the cache
func (h *dbHandler) streamingSetToCache(ctx context.Context, t *streaming.Streaming) error {
	if err := h.cache.StreamingSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// activeflowGetFromCache returns streaming from the cache if possible.
func (h *dbHandler) streamingGetFromCache(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {

	// get from cache
	res, err := h.cache.StreamingGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// StreamingGet returns streaming.
func (h *dbHandler) StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {

	return h.streamingGetFromCache(ctx, id)
}

// StreamingUpdate updates the streaming.
func (h *dbHandler) StreamingUpdate(ctx context.Context, t *streaming.Streaming) error {
	if err := h.streamingSetToCache(ctx, t); err != nil {
		return err
	}

	return nil
}
