package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/variable"
)

type handler struct {
	Addr     string
	Password string
	DB       int

	Cache  *redis.Client
	Locker *redsync.Redsync

	mapMutex map[string]*redsync.Mutex
}

// CacheHandler interface
type CacheHandler interface {
	Connect() error

	ActiveflowGet(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	ActiveflowSet(ctx context.Context, cf *activeflow.Activeflow) error
	ActiveflowGetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	ActiveflowReleaseLock(ctx context.Context, id uuid.UUID) error

	FlowDel(ctx context.Context, id uuid.UUID) error
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowSet(ctx context.Context, flow *flow.Flow) error

	VariableSet(ctx context.Context, t *variable.Variable) error
	VariableGet(ctx context.Context, id uuid.UUID) (*variable.Variable, error)
}

// NewHandler creates DBHandler
func NewHandler(addr string, password string, db int) CacheHandler {

	cache := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// redis lock
	pool := goredis.NewPool(cache)
	locker := redsync.New(pool)

	h := &handler{
		Addr:     addr,
		Password: password,
		DB:       db,
		Cache:    cache,
		Locker:   locker,

		mapMutex: make(map[string]*redsync.Mutex),
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
