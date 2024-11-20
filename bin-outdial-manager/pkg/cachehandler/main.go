package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
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

	OutdialGet(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error)
	OutdialSet(ctx context.Context, t *outdial.Outdial) error

	OutdialTargetGet(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error)
	OutdialTargetSet(ctx context.Context, t *outdialtarget.OutdialTarget) error

	OutdialTargetCallGet(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallSet(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error
	OutdialTargetCallSetByActiveflowID(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error
	OutdialTargetCallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error)
	OutdialTargetCallSetByReferenceID(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error
	OutdialTargetCallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error)
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
