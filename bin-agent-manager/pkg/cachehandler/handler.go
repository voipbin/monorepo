package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"
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
	key := fmt.Sprintf("agent:agent:%d", id)

	var res agent.Agent
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentrSet sets the agent info into the cache.
func (h *handler) AgentSet(ctx context.Context, u *agent.Agent) error {
	key := fmt.Sprintf("agent:agent:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// ResourceGet returns cached resource info
func (h *handler) ResourceGet(ctx context.Context, id uuid.UUID) (*resource.Resource, error) {
	key := fmt.Sprintf("agent:resource:%d", id)

	var res resource.Resource
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ResourceSet sets the resource info into the cache.
func (h *handler) ResourceSet(ctx context.Context, u *resource.Resource) error {
	key := fmt.Sprintf("agent:resource:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}
