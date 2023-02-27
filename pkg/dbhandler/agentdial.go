package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
)

// AgentSetToCache sets the given agent to the cache
func (h *handler) AgentDialSetToCache(ctx context.Context, u *agentdial.AgentDial) error {
	if err := h.cache.AgentDialSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// AgentDialGetFromCache returns agent from the cache.
func (h *handler) AgentDialGetFromCache(ctx context.Context, id uuid.UUID) (*agentdial.AgentDial, error) {

	// get from cache
	res, err := h.cache.AgentDialGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *handler) AgentDialGet(ctx context.Context, id uuid.UUID) (*agentdial.AgentDial, error) {
	return h.AgentDialGetFromCache(ctx, id)
}

func (h *handler) AgentDialCreate(ctx context.Context, a *agentdial.AgentDial) error {
	return h.AgentDialSetToCache(ctx, a)
}
