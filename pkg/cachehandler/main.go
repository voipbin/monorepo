package cachehandler

//go:generate mockgen -destination ./mock_cachehandler_cachehandler.go -package cachehandler -source ./main.go CacheHandler

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
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

	AstAORDel(ctx context.Context, id string) error
	AstAORGet(ctx context.Context, id string) (*models.AstAOR, error)
	AstAORSet(ctx context.Context, e *models.AstAOR) error

	AstAuthDel(ctx context.Context, id string) error
	AstAuthGet(ctx context.Context, id string) (*models.AstAuth, error)
	AstAuthSet(ctx context.Context, e *models.AstAuth) error

	AstEndpointDel(ctx context.Context, id string) error
	AstEndpointGet(ctx context.Context, id string) (*models.AstEndpoint, error)
	AstEndpointSet(ctx context.Context, e *models.AstEndpoint) error

	DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainSet(ctx context.Context, e *models.Domain) error
	DomainDel(ctx context.Context, id uuid.UUID) error

	ExtensionGet(ctx context.Context, id uuid.UUID) (*models.Extension, error)
	ExtensionSet(ctx context.Context, e *models.Extension) error
	ExtensionDel(ctx context.Context, id uuid.UUID) error
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
