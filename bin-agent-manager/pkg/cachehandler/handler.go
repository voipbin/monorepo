package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
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

// AgentGet returns cached agent info
func (h *handler) AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error) {
	key := fmt.Sprintf("agent:%d", id)

	var res agent.Agent
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentrSet sets the agent info into the cache.
func (h *handler) AgentSet(ctx context.Context, u *agent.Agent) error {
	key := fmt.Sprintf("agent:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}
