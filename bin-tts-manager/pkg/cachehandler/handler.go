package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-tts-manager/models/streaming"
)

// getSerialize returns cached serialized info.
func (h *handler) getSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := h.Cache.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(tmp), &data); err != nil {
		return err
	}

	return nil
}

// setSerialize sets the info into the cache after serialization.
func (h *handler) setSerialize(ctx context.Context, key string, data interface{}) error {
	tmp, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, time.Hour*24).Err(); err != nil {
		return err
	}
	return nil
}

// StreamingSet sets the streaming info into the cache.
func (h *handler) StreamingSet(ctx context.Context, s *streaming.Streaming) error {
	key := fmt.Sprintf("tts:streaming:%s", s.ID)

	if err := h.setSerialize(ctx, key, s); err != nil {
		return err
	}

	return nil
}

// StreamingGet returns cached streaming info
func (h *handler) StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	key := fmt.Sprintf("tts:streaming:%s", id)

	var res streaming.Streaming
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
