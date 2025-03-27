package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
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

	AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	AISet(ctx context.Context, data *ai.AI) error

	AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	AIcallSet(ctx context.Context, data *aicall.AIcall) error

	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageSet(ctx context.Context, data *message.Message) error

	SummaryGet(ctx context.Context, id uuid.UUID) (*summary.Summary, error)
	SummarySet(ctx context.Context, data *summary.Summary) error
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
