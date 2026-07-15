package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"monorepo/bin-email-manager/models/email"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache  *redis.Client
	Locker *redsync.Redsync

	mapMutex map[string]*redsync.Mutex
}

// CacheHandler interface
type CacheHandler interface {
	Connect() error

	EmailGet(ctx context.Context, id uuid.UUID) (*email.Email, error)
	EmailSet(ctx context.Context, e *email.Email) error

	// RateLimitIncrement atomically increments the counter at key (INCR) and
	// unconditionally attempts to set a TTL on it via EXPIRE...NX (only applies
	// if the key has no TTL yet). Unconditional ExpireNX (rather than gating on
	// count==1) avoids a permanent-lockout gap if the process crashes between
	// INCR and EXPIRE (e.g. pod restart during a rolling deploy). VOIP-1259.
	RateLimitIncrement(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// NewHandler creates DBHandler
func NewHandler(addr string, password string, db int) CacheHandler {

	cache := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// redis lock
	pool := goredis.NewPool(cache)
	locker := redsync.New(pool)

	h := &handler{
		Addr:     addr,
		Password: password,
		DB:       db,
		Cache:    cache,
		Locker:   locker,

		mapMutex: make(map[string]*redsync.Mutex),
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
