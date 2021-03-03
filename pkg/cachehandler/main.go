package cachehandler

//go:generate mockgen -destination ./mock_cachehandler_cachehandler.go -package cachehandler gitlab.com/voipbin/bin-manager/api-manager.git/pkg/cachehandler CacheHandler

import (
	"context"

	"github.com/go-redis/redis/v8"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
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

	UserGet(ctx context.Context, id uint64) (*models.User, error)
	UserSet(ctx context.Context, u *models.User) error
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
