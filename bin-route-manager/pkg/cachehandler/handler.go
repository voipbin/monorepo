package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/route"
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

// ProviderSet sets the provider info into the cache.
func (h *handler) ProviderSet(ctx context.Context, data *provider.Provider) error {
	key := fmt.Sprintf("provider:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// ProviderGet returns cached provider info
func (h *handler) ProviderGet(ctx context.Context, id uuid.UUID) (*provider.Provider, error) {
	key := fmt.Sprintf("provider:%s", id)

	var res provider.Provider
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteSet sets the route info into the cache.
func (h *handler) RouteSet(ctx context.Context, data *route.Route) error {
	key := fmt.Sprintf("route:%s", data.ID)

	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	return nil
}

// RouteGet returns cached route info
func (h *handler) RouteGet(ctx context.Context, id uuid.UUID) (*route.Route, error) {
	key := fmt.Sprintf("route:%s", id)

	var res route.Route
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RouteDelete deletes cached route info
func (h *handler) RouteDelete(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("route:%s", id)

	if errDel := h.Cache.Del(ctx, key); errDel.Err() != nil {
		return errDel.Err()
	}

	return nil
}
