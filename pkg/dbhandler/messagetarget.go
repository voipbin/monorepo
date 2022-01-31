package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
)

// messageTargetGetFromCache returns messagetarget from the cache.
func (h *handler) messageTargetGetFromCache(ctx context.Context, id uuid.UUID) (*messagetarget.MessageTarget, error) {

	// get from cache
	res, err := h.cache.MessageTargetGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MessageTargetSetToCache sets the given agent to the cache
func (h *handler) messageTargetSetToCache(ctx context.Context, u *messagetarget.MessageTarget) error {
	if err := h.cache.MessageTargetSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// MessageTargetGet returns the messagetarget.
func (h *handler) MessageTargetGet(ctx context.Context, id uuid.UUID) (*messagetarget.MessageTarget, error) {
	return h.messageTargetGetFromCache(ctx, id)
}

// MessageTargetSet returns sets the messagetarget
func (h *handler) MessageTargetSet(ctx context.Context, u *messagetarget.MessageTarget) error {
	return h.messageTargetSetToCache(ctx, u)
}
