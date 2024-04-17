package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"

	"monorepo/bin-transfer-manager/models/transfer"
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

	TransferGet(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error)
	TransferGetByTransfererCallID(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error)
	TransferGetByGroupcallID(ctx context.Context, groupcallID uuid.UUID) (*transfer.Transfer, error)
	TransferSet(ctx context.Context, tr *transfer.Transfer) error
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
