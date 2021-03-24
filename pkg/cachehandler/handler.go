package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
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

// UserGet returns cached user info
func (h *handler) UserGet(ctx context.Context, id uint64) (*user.User, error) {
	key := fmt.Sprintf("user:%d", id)

	var res user.User
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// UserSet sets the user info into the cache.
func (h *handler) UserSet(ctx context.Context, u *user.User) error {
	key := fmt.Sprintf("user:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}
