package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_cachehandler_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
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
	AccountGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error)
	AccountSet(ctx context.Context, data *account.Account) error

	BillingGet(ctx context.Context, id uuid.UUID) (*billing.Billing, error)
	BillingGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error)
	BillingSet(ctx context.Context, data *billing.Billing) error
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
