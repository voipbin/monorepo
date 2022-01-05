package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentcall"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
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

// AgentDialGet returns cached agentdial info
func (h *handler) AgentDialGet(ctx context.Context, id uuid.UUID) (*agentdial.AgentDial, error) {
	key := fmt.Sprintf("agentdial:%d", id)

	var res agentdial.AgentDial
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentDialSet sets the agentdial info into the cache.
func (h *handler) AgentDialSet(ctx context.Context, u *agentdial.AgentDial) error {
	key := fmt.Sprintf("agentdial:%d", u.AgentID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// AgentCallGet returns cached agentcall info
func (h *handler) AgentCallGet(ctx context.Context, id uuid.UUID) (*agentcall.AgentCall, error) {
	key := fmt.Sprintf("agentcall:%d", id)

	var res agentcall.AgentCall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// AgentCallSet sets the agentcall info into the cache.
func (h *handler) AgentCallSet(ctx context.Context, u *agentcall.AgentCall) error {
	key := fmt.Sprintf("agentcall:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// TagGet returns cached tag info
func (h *handler) TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	key := fmt.Sprintf("tag:%d", id)

	var res tag.Tag
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TagSet sets the tag info into the cache.
func (h *handler) TagSet(ctx context.Context, u *tag.Tag) error {
	key := fmt.Sprintf("tag:%d", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}
