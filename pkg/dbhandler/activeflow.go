package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/activeflow"
)

// ActiveFlowSetToCache sets the given callflow to the cache
func (h *handler) ActiveFlowSetToCache(ctx context.Context, flow *activeflow.ActiveFlow) error {
	if err := h.cache.ActiveFlowSet(ctx, flow); err != nil {
		return err
	}

	return nil
}

// ActiveFlowGetFromCache returns flow from the cache if possible.
func (h *handler) ActiveFlowGetFromCache(ctx context.Context, id uuid.UUID) (*activeflow.ActiveFlow, error) {

	// get from cache
	res, err := h.cache.ActiveFlowGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (h *handler) ActiveFlowCreate(ctx context.Context, af *activeflow.ActiveFlow) error {

	if err := h.ActiveFlowSetToCache(ctx, af); err != nil {
		return err
	}

	return nil
}

// ActiveFlowGet returns activeflow.
func (h *handler) ActiveFlowGet(ctx context.Context, id uuid.UUID) (*activeflow.ActiveFlow, error) {

	res, err := h.ActiveFlowGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	return res, nil
}

// ActiveFlowSet sets the activeflow.
func (h *handler) ActiveFlowSet(ctx context.Context, af *activeflow.ActiveFlow) error {

	if err := h.ActiveFlowSetToCache(ctx, af); err != nil {
		return err
	}

	return nil
}
