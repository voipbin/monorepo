package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
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

	AccesskeyGet(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error)
	AccesskeySet(ctx context.Context, a *accesskey.Accesskey) error

	CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	CustomerSet(ctx context.Context, c *customer.Customer) error

	EmailVerifyTokenSet(ctx context.Context, token string, customerID uuid.UUID, ttl time.Duration) error
	EmailVerifyTokenGet(ctx context.Context, token string) (uuid.UUID, error)
	EmailVerifyTokenDelete(ctx context.Context, token string) error

	SignupSessionSet(ctx context.Context, tempToken string, session *SignupSession, ttl time.Duration) error
	SignupSessionGet(ctx context.Context, tempToken string) (*SignupSession, error)
	SignupSessionDelete(ctx context.Context, tempToken string) error
	SignupAttemptIncrement(ctx context.Context, tempToken string, ttl time.Duration) (int64, error)
	SignupAttemptDelete(ctx context.Context, tempToken string) error
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
