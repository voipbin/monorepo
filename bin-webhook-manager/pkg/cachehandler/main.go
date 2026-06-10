package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-webhook-manager/models/account"
	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
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

	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountSet(ctx context.Context, u *account.Account) error

	ActiveflowWebhookGet(ctx context.Context, id uuid.UUID) (*mwactiveflow.Webhook, bool, error)
	ActiveflowWebhookSet(ctx context.Context, id uuid.UUID, w *mwactiveflow.Webhook, ttl time.Duration) error
	ActiveflowWebhookSetNegative(ctx context.Context, id uuid.UUID, tm time.Time, tmDelete *time.Time, ttl time.Duration) error
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
