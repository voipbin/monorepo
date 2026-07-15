package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/message"
)

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache *redis.Client
}

// CacheHandler interface
type CacheHandler interface {
	Connect() error

	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageSet(ctx context.Context, m *message.Message) error

	// RateLimitIncrement atomically increments the counter at key (INCR) and
	// unconditionally attempts to set a TTL on it via EXPIRE...NX (only takes effect
	// if the key has no TTL yet). Returns the counter's new value after the increment.
	// Calling ExpireNX unconditionally on every request (instead of only when
	// count==1) closes a permanent-lockout gap: if the process crashes between INCR
	// and EXPIRE (e.g. pod restart during a rolling deploy), the key would otherwise
	// keep incrementing forever with no TTL. Requires Redis 7.0+ (ExpireNX). VOIP-1259.
	RateLimitIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// NewHandler creates DBHandler
func NewHandler(addr string, password string, db int) CacheHandler {

	cache := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	h := &handler{
		Addr:     addr,
		Password: password,
		DB:       db,
		Cache:    cache,
	}

	return h
}

// Connect connects to the cache server
func (h *handler) Connect() error {
	_, err := h.Cache.Ping(context.Background()).Result()
	if err != nil {
		return err
	}

	return nil
}
