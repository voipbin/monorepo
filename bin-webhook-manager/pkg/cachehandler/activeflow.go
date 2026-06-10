package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
)

// activeflowWebhookKey returns the Redis key for a per-activeflow webhook entry.
func activeflowWebhookKey(id uuid.UUID) string {
	return fmt.Sprintf("webhook:activeflow:%s", id)
}

// ActiveflowWebhookGet returns the cached per-activeflow webhook entry.
//
// The second return value reports whether an entry exists in the cache. A miss
// (key absent) returns (nil, false, nil); a real Redis error returns
// (nil, false, err).
func (h *handler) ActiveflowWebhookGet(ctx context.Context, id uuid.UUID) (*mwactiveflow.Webhook, bool, error) {
	key := activeflowWebhookKey(id)

	tmp, err := h.Cache.Get(ctx, key).Result()
	if err == redis.Nil {
		// cache miss
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var res mwactiveflow.Webhook
	if errUnmarshal := json.Unmarshal([]byte(tmp), &res); errUnmarshal != nil {
		return nil, false, errUnmarshal
	}

	return &res, true, nil
}

// activeflowWebhookSetRaw writes the entry honoring monotonic ordering: a write
// only applies when its timestamp is not older than the stored entry's
// timestamp (design 5.6). It returns nil when the write is skipped due to a
// newer existing entry.
func (h *handler) activeflowWebhookSetRaw(ctx context.Context, id uuid.UUID, w *mwactiveflow.Webhook, ttl time.Duration) error {
	key := activeflowWebhookKey(id)

	// monotonic guard: do not let an older event overwrite a newer one.
	existing, found, err := h.ActiveflowWebhookGet(ctx, id)
	if err == nil && found && existing != nil && w.Tm.Before(existing.Tm) {
		// stored entry is newer; skip this write.
		return nil
	}

	tmp, err := json.Marshal(w)
	if err != nil {
		return err
	}

	if err := h.Cache.Set(ctx, key, tmp, ttl).Err(); err != nil {
		return err
	}

	return nil
}

// ActiveflowWebhookSet stores a positive per-activeflow webhook entry with the
// given TTL, honoring monotonic ordering.
func (h *handler) ActiveflowWebhookSet(ctx context.Context, id uuid.UUID, w *mwactiveflow.Webhook, ttl time.Duration) error {
	if w == nil {
		return fmt.Errorf("nil webhook entry")
	}
	return h.activeflowWebhookSetRaw(ctx, id, w, ttl)
}

// ActiveflowWebhookSetNegative stores a negative tombstone entry (no webhook,
// deleted, or transient) with the given TTL, honoring monotonic ordering. The
// optional tmDelete carries the delete timestamp for delete tombstones.
func (h *handler) ActiveflowWebhookSetNegative(ctx context.Context, id uuid.UUID, tm time.Time, tmDelete *time.Time, ttl time.Duration) error {
	w := &mwactiveflow.Webhook{
		Deleted:  true,
		TMDelete: tmDelete,
		Tm:       tm,
	}
	return h.activeflowWebhookSetRaw(ctx, id, w, ttl)
}
