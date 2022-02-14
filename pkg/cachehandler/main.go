package cachehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentcall"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
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

	AgentCallGet(ctx context.Context, id uuid.UUID) (*agentcall.AgentCall, error)
	AgentCallSet(ctx context.Context, u *agentcall.AgentCall) error

	AgentDialGet(ctx context.Context, id uuid.UUID) (*agentdial.AgentDial, error)
	AgentDialSet(ctx context.Context, u *agentdial.AgentDial) error

	TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	TagSet(ctx context.Context, u *tag.Tag) error
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
