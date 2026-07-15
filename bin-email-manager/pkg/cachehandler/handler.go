package cachehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-email-manager/models/email"
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

// EmailSet sets the email info into the cache.
func (h *handler) EmailSet(ctx context.Context, e *email.Email) error {
	key := fmt.Sprintf("email_email:%s", e.ID)

	if err := h.setSerialize(ctx, key, e); err != nil {
		return err
	}

	return nil
}

// EmailGet returns cached email info
func (h *handler) EmailGet(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	key := fmt.Sprintf("email_email:%s", id)

	var res email.Email
	if err := h.getSerialize(ctx, key, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// RateLimitIncrement atomically increments the counter at key (INCR) and
// unconditionally attempts to set a TTL on it via EXPIRE...NX. See interface
// doc comment in main.go for the crash-safety rationale. VOIP-1259.
func (h *handler) RateLimitIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	count, err := h.Cache.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// Unconditionally attempt to set TTL with NX (only applies if the key has no
	// TTL yet). Safe to call on every request — a key that already has a TTL is a
	// no-op. Log-and-continue: a transient EXPIRE failure does not invalidate the
	// INCR result, and the next request retries ExpireNX regardless of count.
	if _, expErr := h.Cache.ExpireNX(ctx, key, ttl).Result(); expErr != nil {
		logrus.WithFields(logrus.Fields{
			"func": "RateLimitIncrement",
			"key":  key,
		}).Errorf("Could not set TTL on rate limit key via ExpireNX. err: %v", expErr)
	}

	return count, nil
}
