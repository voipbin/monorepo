package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/address"
	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/models/resource"
	commonaddress "monorepo/bin-common-handler/models/address"
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

	AddressGetByCommonAddress(ctx context.Context, u commonaddress.Address) (*address.Address, error)
	AddressSet(ctx context.Context, u *address.Address) error
	AddressDel(ctx context.Context, u *address.Address) error

	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentSet(ctx context.Context, u *agent.Agent) error

	ResourceGet(ctx context.Context, id uuid.UUID) (*resource.Resource, error)
	ResourceSet(ctx context.Context, u *resource.Resource) error
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
