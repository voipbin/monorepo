package cachehandler

//go:generate mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
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

	StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
	StreamingSet(ctx context.Context, stream *streaming.Streaming) error

	TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	TranscribeSet(ctx context.Context, trans *transcribe.Transcribe) error

	TranscriptGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error)
	TranscriptSet(ctx context.Context, trans *transcript.Transcript) error
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
