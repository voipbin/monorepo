package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
	"gitlab.com/voipbin/bin-manager/route-manager.git/models/route"
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

	ProviderSet(ctx context.Context, data *provider.Provider) error
	ProviderGet(ctx context.Context, id uuid.UUID) (*provider.Provider, error)

	RouteSet(ctx context.Context, data *route.Route) error
	RouteGet(ctx context.Context, id uuid.UUID) (*route.Route, error)
	RouteDelete(ctx context.Context, id uuid.UUID) error
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
