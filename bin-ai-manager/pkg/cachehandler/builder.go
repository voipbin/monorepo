package cachehandler

import (
	"context"
	"fmt"
	"time"

	uuid "github.com/gofrs/uuid"
)

// builderChatCountKey is the per-customer daily counter of Assistant Builder
// turns (design 4.3). It is NOT a listen key and does not share that prefix.
// The key carries the customer id only, never any conversation text.
func builderChatCountKey(customerID uuid.UUID) string {
	return fmt.Sprintf("ai:builder:chat:count:%s", customerID)
}

// builderChatCountScript: INCR the counter and arm the expiry only when the key
// has none. Arming only when TTL < 0 (no expiry) is deliberate and differs from
// listenIncrExpireScript: that counter is re-armed on every call because it
// tracks activity, while this one is a fixed-window limit, and re-arming it on
// every call would let a customer who keeps calling push the window end away
// for ever. TTL returns -1 for a key with no expiry (INCR has just created the key if it
// was missing, so -2 cannot occur here), so a counter that lost its TTL heals on its
// next increment instead of becoming a permanent lock-out.
//
// INCR and EXPIRE are one script so neither can land without the other.
const builderChatCountScript = `local n = redis.call("INCR",KEYS[1]) if redis.call("TTL",KEYS[1]) < 0 then redis.call("EXPIRE",KEYS[1],ARGV[1]) end return n`

// BuilderChatCountIncr counts one Assistant Builder turn for the customer and
// returns the new count. The caller compares it with the daily limit.
func (h *handler) BuilderChatCountIncr(ctx context.Context, customerID uuid.UUID, ttl time.Duration) (int64, error) {
	return h.Cache.Eval(ctx, builderChatCountScript,
		[]string{builderChatCountKey(customerID)},
		listenTTLSeconds(ttl),
	).Int64()
}
