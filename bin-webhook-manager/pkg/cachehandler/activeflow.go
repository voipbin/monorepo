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

// activeflowWebhookCASScript performs an atomic compare-and-set honoring
// monotonic ordering (design 5.6) in a single Redis round trip:
//
//   - KEYS[1] = cache key
//   - ARGV[1] = new value (JSON)
//   - ARGV[2] = incoming Tm in unix-nano
//   - ARGV[3] = TTL in milliseconds
//
// It reads the stored entry's Tm and only overwrites when the incoming Tm is
// not older than the stored one (or the key is absent), guarding against the
// created-after-deleted resurrection race.
const activeflowWebhookCASScript = `
local existing = redis.call('GET', KEYS[1])
if existing then
	local ok, decoded = pcall(cjson.decode, existing)
	if ok and decoded ~= nil and decoded['tm'] ~= nil then
		local storedTm = tonumber(decoded['tm'])
		if storedTm ~= nil and tonumber(ARGV[2]) < storedTm then
			-- stored entry is newer; skip this write.
			return 0
		end
	end
end
redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[3])
return 1
`

// activeflowWebhookSetRaw writes the entry honoring monotonic ordering: a write
// only applies when its timestamp is not older than the stored entry's
// timestamp (design 5.6). The read+compare+set runs as an atomic Lua script in
// a single round trip, so concurrent writers cannot race. It returns nil when
// the write is skipped due to a newer existing entry.
func (h *handler) activeflowWebhookSetRaw(ctx context.Context, id uuid.UUID, w *mwactiveflow.Webhook, ttl time.Duration) error {
	key := activeflowWebhookKey(id)

	tmp, err := json.Marshal(w)
	if err != nil {
		return err
	}

	// The Lua script compares the incoming Tm (unix-nano) against the stored
	// entry's tm and only overwrites when the incoming write is not older.
	// Note: the JSON 'tm' field must be a unix-nano integer for the script to
	// compare it, so we pass it explicitly as ARGV[2].
	incomingTm := w.Tm.UnixNano()
	ttlMS := ttl.Milliseconds()

	if errEval := h.Cache.Eval(
		ctx,
		activeflowWebhookCASScript,
		[]string{key},
		tmp,
		incomingTm,
		ttlMS,
	).Err(); errEval != nil && errEval != redis.Nil {
		return errEval
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
