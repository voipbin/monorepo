package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
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

// OutdialSet sets the outdial info into the cache.
func (h *handler) OutdialSet(ctx context.Context, t *outdial.Outdial) error {
	key := fmt.Sprintf("outdial:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// OutdialGet returns cached outdial info
func (h *handler) OutdialGet(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error) {
	key := fmt.Sprintf("outdial:%s", id)

	var res outdial.Outdial
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialTargetSet sets the outdialtarget info into the cache.
func (h *handler) OutdialTargetSet(ctx context.Context, t *outdialtarget.OutdialTarget) error {
	key := fmt.Sprintf("outdialtarget:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// OutdialTargetGet returns cached outdialtarget info
func (h *handler) OutdialTargetGet(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error) {
	key := fmt.Sprintf("outdialtarget:%s", id)

	var res outdialtarget.OutdialTarget
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialTargetCallSet sets the outdialtargetcall info into the cache.
func (h *handler) OutdialTargetCallSet(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {
	key := fmt.Sprintf("outdialtargetcall:%s", t.ID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// OutdialTargetCallGet returns cached outdialtargetcall info
func (h *handler) OutdialTargetCallGet(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {
	key := fmt.Sprintf("outdialtargetcall:%s", id)

	var res outdialtargetcall.OutdialTargetCall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialTargetCallSetByActiveflowID sets the outdialtargetcall info into the cache.
func (h *handler) OutdialTargetCallSetByActiveflowID(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {
	key := fmt.Sprintf("outdialtargetcall_activeflowid:%s", t.ActiveflowID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// OutdialTargetCallGet returns cached outdialtargetcall info
func (h *handler) OutdialTargetCallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {
	key := fmt.Sprintf("outdialtargetcall_activeflowid:%s", activeflowID)

	var res outdialtargetcall.OutdialTargetCall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialTargetCallSetByReferenceID sets the outdialtargetcall info into the cache.
func (h *handler) OutdialTargetCallSetByReferenceID(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {
	key := fmt.Sprintf("outdialtargetcall_referenceid:%s", t.ReferenceID)

	if err := h.setSerialize(ctx, key, t); err != nil {
		return err
	}

	return nil
}

// OutdialTargetCallGetByReferenceID returns cached outdialtargetcall info
func (h *handler) OutdialTargetCallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {
	key := fmt.Sprintf("outdialtargetcall_referenceid:%s", referenceID)

	var res outdialtargetcall.OutdialTargetCall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
