package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
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
	AstAORGet(ctx context.Context, id string) (*astaor.AstAOR, error)
	AstAORSet(ctx context.Context, e *astaor.AstAOR) error

	AstAuthDel(ctx context.Context, id string) error
	AstAuthGet(ctx context.Context, id string) (*astauth.AstAuth, error)
	AstAuthSet(ctx context.Context, e *astauth.AstAuth) error

	AstContactsDel(ctx context.Context, endpoint string) error
	AstContactsGet(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error)
	AstContactsSet(ctx context.Context, endpoint string, contacts []*astcontact.AstContact) error

	AstEndpointDel(ctx context.Context, id string) error
	AstEndpointGet(ctx context.Context, id string) (*astendpoint.AstEndpoint, error)
	AstEndpointSet(ctx context.Context, e *astendpoint.AstEndpoint) error

	DomainGet(ctx context.Context, id uuid.UUID) (*domain.Domain, error)
	DomainSet(ctx context.Context, e *domain.Domain) error
	DomainDel(ctx context.Context, id uuid.UUID) error

	ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	ExtensionSet(ctx context.Context, e *extension.Extension) error
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
