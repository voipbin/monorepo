// Package ratelimithandler wraps github.com/go-redis/redis_rate/v9 (a Redis
// GCRA rate limiter) behind a local interface, so the rest of the codebase
// never imports the vendor package directly.
//
// Rationale (VOIP-1302 design §4-13): the monorepo has a documented exit
// strategy for go-redis v8 -> v9 that will require redis_rate v9 -> v10 in
// lockstep (see design doc §4-2). Keeping the vendor type behind a local
// interface confines that future migration to this package instead of
// every call site.
package ratelimithandler

//go:generate mockgen -package ratelimithandler -destination ./mock_ratelimithandler_ratelimithandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	redisrate "github.com/go-redis/redis_rate/v9"
)

// Limit describes a GCRA rate limit: Rate events allowed per Period, with a
// maximum burst of Burst events.
type Limit struct {
	Rate   int
	Burst  int
	Period time.Duration
}

// Result is the outcome of a rate limit check.
type Result struct {
	// Allowed is the number of events permitted by this call (0 or 1 for a
	// single-event Allow check).
	Allowed int
	// Remaining is the number of additional events that could be permitted
	// instantaneously for this key given the current state.
	Remaining int
	// RetryAfter is the time until the next request will be permitted. It
	// is -1 when the request was allowed.
	RetryAfter time.Duration
}

// RateLimiter checks and consumes rate limit budget for a key.
type RateLimiter interface {
	// Allow reports whether a single event for key is permitted under limit,
	// consuming budget if so.
	Allow(ctx context.Context, key string, limit Limit) (*Result, error)
}

type handler struct {
	limiter *redisrate.Limiter
}

// NewHandler creates a RateLimiter backed by the given Redis client.
func NewHandler(client *redis.Client) RateLimiter {
	return &handler{
		limiter: redisrate.NewLimiter(client),
	}
}

// Allow implements RateLimiter.
func (h *handler) Allow(ctx context.Context, key string, limit Limit) (*Result, error) {
	res, err := h.limiter.Allow(ctx, key, redisrate.Limit{
		Rate:   limit.Rate,
		Burst:  limit.Burst,
		Period: limit.Period,
	})
	if err != nil {
		return nil, err
	}

	return &Result{
		Allowed:    res.Allowed,
		Remaining:  res.Remaining,
		RetryAfter: res.RetryAfter,
	}, nil
}
