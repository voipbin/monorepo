package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
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

// FlowSet sets the flow info into the cache.
func (h *handler) FlowSet(ctx context.Context, flow *flow.Flow) error {
	key := fmt.Sprintf("flow:%s", flow.ID)

	if err := h.setSerialize(ctx, key, flow); err != nil {
		return err
	}

	return nil
}

// FlowGet returns cached flow info
func (h *handler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	key := fmt.Sprintf("flow:%s", id)

	var res flow.Flow
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
