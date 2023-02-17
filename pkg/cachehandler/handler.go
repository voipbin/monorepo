package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
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

// delSerialize deletes the cached serialized data.
func (h *handler) delSerialize(ctx context.Context, key string) error {
	if err := h.Cache.Del(ctx, key).Err(); err != nil {
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

// FlowDel returns cached flow info
func (h *handler) FlowDel(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("flow:%s", id)

	if err := h.delSerialize(ctx, key); err != nil {
		return err
	}

	return nil
}

// ActiveflowSet sets the activeflow info into the cache
func (h *handler) ActiveflowSet(ctx context.Context, af *activeflow.Activeflow) error {
	key := fmt.Sprintf("activeflow:%s", af.ID)

	if err := h.setSerialize(ctx, key, af); err != nil {
		return err
	}

	return nil
}

// ActiveflowGet returns cached activeflow info
func (h *handler) ActiveflowGet(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	key := fmt.Sprintf("activeflow:%s", id)

	var res activeflow.Activeflow
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ActiveflowGet returns cached activeflow info
func (h *handler) ActiveflowGetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	keyLock := fmt.Sprintf("activeflow:lock:%s", id)
	mutex := h.Locker.NewMutex(keyLock)
	if errLock := mutex.Lock(); errLock != nil {
		return nil, errLock
	}

	key := fmt.Sprintf("activeflow:%s", id)
	var res activeflow.Activeflow
	if err := h.getSerialize(ctx, key, &res); err != nil {
		_, _ = mutex.Unlock()
		return nil, err
	}

	h.mapMutex[id.String()] = mutex
	return &res, nil
}

// ActiveflowReleaseLock returns cached activeflow info
func (h *handler) ActiveflowReleaseLock(ctx context.Context, id uuid.UUID) error {

	mutex, ok := h.mapMutex[id.String()]
	if !ok {
		return fmt.Errorf("no mutex")
	}
	defer delete(h.mapMutex, id.String())

	_, err := mutex.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// ActiveflowSet sets the variable info into the cache
func (h *handler) VariableSet(ctx context.Context, t *variable.Variable) error {
	key := fmt.Sprintf("variable:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// VariableGet returns cached variable info
func (h *handler) VariableGet(ctx context.Context, id uuid.UUID) (*variable.Variable, error) {
	key := fmt.Sprintf("variable:%s", id)

	var res variable.Variable
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
