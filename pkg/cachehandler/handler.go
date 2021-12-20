package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
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

// QueueGet returns cached agent info
func (h *handler) QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	key := fmt.Sprintf("queue:%s", id)

	var res queue.Queue
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueueSet sets the agent info into the cache.
func (h *handler) QueueSet(ctx context.Context, u *queue.Queue) error {
	key := fmt.Sprintf("queue:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// QueuecallGet returns cached queuecall info
func (h *handler) QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	key := fmt.Sprintf("queuecall:%s", id)

	var res queuecall.Queuecall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueuecallSet sets the queuecall info into the cache.
func (h *handler) QueuecallSet(ctx context.Context, u *queuecall.Queuecall) error {
	key := fmt.Sprintf("queuecall:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// QueuecallReferenceGet returns cached queuecall info
func (h *handler) QueuecallReferenceGet(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error) {
	key := fmt.Sprintf("queuecallreference:%s", id)

	var res queuecallreference.QueuecallReference
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueuecallReferenceSet sets the queuecall info into the cache.
func (h *handler) QueuecallReferenceSet(ctx context.Context, u *queuecallreference.QueuecallReference) error {
	key := fmt.Sprintf("queuecallreference:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}
