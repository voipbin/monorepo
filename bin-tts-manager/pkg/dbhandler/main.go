package dbhandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
)

type DBHandler interface {
	StreamingCreate(ctx context.Context, s *streaming.Streaming) error
	StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
	StreamingUpdate(ctx context.Context, s *streaming.Streaming) error
}

type dbHandler struct {
	util  utilhandler.UtilHandler
	cache cachehandler.CacheHandler
}

func NewDBHandler(cache cachehandler.CacheHandler) DBHandler {
	return &dbHandler{
		util:  utilhandler.NewUtilHandler(),
		cache: cache,
	}
}
