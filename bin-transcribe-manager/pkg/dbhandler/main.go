package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {

	// streaming
	StreamingCreate(ctx context.Context, s *streaming.Streaming) error
	StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)

	// transcribe
	TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error
	TranscribeDelete(ctx context.Context, id uuid.UUID) error
	TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	TranscribeGetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error)
	TranscribeList(ctx context.Context, size uint64, token string, filters map[transcribe.Field]any) ([]*transcribe.Transcribe, error)
	TranscribeUpdate(ctx context.Context, id uuid.UUID, fields map[transcribe.Field]any) error

	// transcript
	TranscriptCreate(ctx context.Context, t *transcript.Transcript) error
	TranscriptGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error)
	TranscriptList(ctx context.Context, size uint64, token string, filters map[transcript.Field]any) ([]*transcript.Transcript, error)
	TranscriptDelete(ctx context.Context, id uuid.UUID) error
	TranscriptUpdate(ctx context.Context, id uuid.UUID, fields map[transcript.Field]any) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}


// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
