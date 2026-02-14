//go:generate mockgen -package dbhandler -destination mock_main.go -source main.go

package dbhandler

import (
	"context"
	"database/sql"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
)

type DBHandler interface {
	StreamingCreate(ctx context.Context, s *streaming.Streaming) error
	StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)
	StreamingUpdate(ctx context.Context, s *streaming.Streaming) error

	SpeakingCreate(ctx context.Context, s *speaking.Speaking) error
	SpeakingGet(ctx context.Context, id uuid.UUID) (*speaking.Speaking, error)
	SpeakingGets(ctx context.Context, token string, size uint64, filters map[speaking.Field]any) ([]*speaking.Speaking, error)
	SpeakingUpdate(ctx context.Context, id uuid.UUID, fields map[speaking.Field]any) error
	SpeakingDelete(ctx context.Context, id uuid.UUID) error
}

type dbHandler struct {
	db    *sql.DB
	util  utilhandler.UtilHandler
	cache cachehandler.CacheHandler
}

func NewDBHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	return &dbHandler{
		db:    db,
		util:  utilhandler.NewUtilHandler(),
		cache: cache,
	}
}
