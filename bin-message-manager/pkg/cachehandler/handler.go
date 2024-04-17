package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/message"
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

// delSerialize deletes cached serialized info.
//
//nolint:unused // this is ok.
func (h *handler) delSerialize(ctx context.Context, key string) error {
	_, err := h.Cache.Del(ctx, key).Result()
	if err != nil {
		return err
	}

	return nil
}

// MessageGet returns message info
func (h *handler) MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	key := fmt.Sprintf("message:%s", id)

	var res message.Message
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// MessageSet sets the message info into the cache.
func (h *handler) MessageSet(ctx context.Context, m *message.Message) error {
	key := fmt.Sprintf("message:%s", m.ID)

	if err := h.setSerialize(ctx, key, m); err != nil {
		return err
	}

	return nil
}
