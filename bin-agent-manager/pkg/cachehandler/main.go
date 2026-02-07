package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
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

	AgentGet(ctx context.Context, id uuid.UUID) (*agent.Agent, error)
	AgentSet(ctx context.Context, u *agent.Agent) error

	PasswordResetTokenSet(ctx context.Context, token string, agentID uuid.UUID, ttl time.Duration) error
	PasswordResetTokenGet(ctx context.Context, token string) (uuid.UUID, error)
	PasswordResetTokenDelete(ctx context.Context, token string) error
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
