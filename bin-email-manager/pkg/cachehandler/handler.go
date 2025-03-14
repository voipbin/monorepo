package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-email-manager/models/email"
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

// EmailSet sets the email info into the cache.
func (h *handler) EmailSet(ctx context.Context, e *email.Email) error {
	key := fmt.Sprintf("email_email:%s", e.ID)

	if err := h.setSerialize(ctx, key, e); err != nil {
		return err
	}

	return nil
}

// EmailGet returns cached email info
func (h *handler) EmailGet(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	key := fmt.Sprintf("email_email:%s", id)

	var res email.Email
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
