package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"time"

	"github.com/gofrs/uuid"
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

func (h *handler) PipecatcallSet(ctx context.Context, pc *pipecatcall.Pipecatcall) error {
	key := fmt.Sprintf("pipecatcall:%s", pc.ID)

	if err := h.setSerialize(ctx, key, pc); err != nil {
		return err
	}

	return nil
}

func (h *handler) PipecatcallGet(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	key := fmt.Sprintf("pipecatcall:%s", id)

	var res pipecatcall.Pipecatcall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
