package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
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

	QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	QueueSet(ctx context.Context, u *queue.Queue) error

	QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallSet(ctx context.Context, u *queuecall.Queuecall) error
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
