package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentcall"
)

// AgentCallSetToCache sets the given agentcall to the cache
func (h *handler) AgentCallSetToCache(ctx context.Context, u *agentcall.AgentCall) error {
	if err := h.cache.AgentCallSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// AgentCallGetFromCache returns agentcall from the cache.
func (h *handler) AgentCallGetFromCache(ctx context.Context, id uuid.UUID) (*agentcall.AgentCall, error) {

	// get from cache
	res, err := h.cache.AgentCallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AgentCallGet
func (h *handler) AgentCallGet(ctx context.Context, id uuid.UUID) (*agentcall.AgentCall, error) {
	return h.AgentCallGetFromCache(ctx, id)
}

// AgentCallCreate
func (h *handler) AgentCallCreate(ctx context.Context, a *agentcall.AgentCall) error {
	return h.AgentCallSetToCache(ctx, a)
}
