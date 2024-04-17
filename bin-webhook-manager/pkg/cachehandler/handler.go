package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/account"
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

// AccountSet sets the account info into the cache.
func (h *handler) AccountSet(ctx context.Context, u *account.Account) error {
	key := fmt.Sprintf("webhook.account:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// AccountGet returns cached account info
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	key := fmt.Sprintf("webhook.account:%d", id)

	var res account.Account
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
