package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/variable"
)

// VariableCreate creates a new variable.
func (h *handler) VariableCreate(ctx context.Context, t *variable.Variable) error {
	return h.variableSetToCache(ctx, t)
}

// variableSetToCache sets the given variable to the cache
func (h *handler) variableSetToCache(ctx context.Context, t *variable.Variable) error {
	if err := h.cache.VariableSet(ctx, t); err != nil {
		return err
	}

	return nil
}

// activeflowGetFromCache returns variable from the cache if possible.
func (h *handler) variableGetFromCache(ctx context.Context, id uuid.UUID) (*variable.Variable, error) {

	// get from cache
	res, err := h.cache.VariableGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// VariableGet returns variable.
func (h *handler) VariableGet(ctx context.Context, id uuid.UUID) (*variable.Variable, error) {

	return h.variableGetFromCache(ctx, id)
}

// VariableUpdate updates the variable.
func (h *handler) VariableUpdate(ctx context.Context, t *variable.Variable) error {
	if err := h.variableSetToCache(ctx, t); err != nil {
		return err
	}

	return nil
}
