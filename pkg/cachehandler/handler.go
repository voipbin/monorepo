package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
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
	key := fmt.Sprintf("queue:queue:%s", id)

	var res queue.Queue
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueueSet sets the agent info into the cache.
func (h *handler) QueueSet(ctx context.Context, u *queue.Queue) error {
	key := fmt.Sprintf("queue:queue:%s", u.ID)

	if err := h.setSerialize(ctx, key, u); err != nil {
		return err
	}

	return nil
}

// QueuecallGet returns cached queuecall info
func (h *handler) QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	key := fmt.Sprintf("queue:queuecall:%s", id)

	var res queuecall.Queuecall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// QueuecallSet sets the queuecall info into the cache.
func (h *handler) QueuecallSet(ctx context.Context, data *queuecall.Queuecall) error {
	key := fmt.Sprintf("queue:queuecall:%s", data.ID)
	if err := h.setSerialize(ctx, key, data); err != nil {
		return err
	}

	keyWithReferenceID := fmt.Sprintf("queue:queuecall:reference_id:%s", data.ID)
	if err := h.setSerialize(ctx, keyWithReferenceID, data); err != nil {
		return err
	}

	return nil
}

// QueuecallGetByReferenceID returns cached queuecall info of the given reference id.
func (h *handler) QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	key := fmt.Sprintf("queue:queuecall:reference_id:%s", referenceID)

	var res queuecall.Queuecall
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
