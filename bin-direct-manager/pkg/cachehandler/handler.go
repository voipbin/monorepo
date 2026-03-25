package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"monorepo/bin-direct-manager/models/direct"
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

// DirectGetByHash returns cached direct info by hash
func (h *handler) DirectGetByHash(ctx context.Context, hash string) (*direct.Direct, error) {
	key := fmt.Sprintf("direct:hash:%s", hash)

	var res direct.Direct
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// DirectSetByHash sets the direct info into the cache by hash.
func (h *handler) DirectSetByHash(ctx context.Context, hash string, d *direct.Direct) error {
	key := fmt.Sprintf("direct:hash:%s", hash)

	if err := h.setSerialize(ctx, key, d); err != nil {
		return err
	}

	return nil
}

// DirectDeleteByHash deletes the direct info from the cache by hash.
func (h *handler) DirectDeleteByHash(ctx context.Context, hash string) error {
	key := fmt.Sprintf("direct:hash:%s", hash)

	if err := h.Cache.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}
