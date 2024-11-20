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
	TranscribeAddTranscript(ctx context.Context, id uuid.UUID, t *transcript.Transcript) error
	TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error
	TranscribeDelete(ctx context.Context, id uuid.UUID) error
	TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	TranscribeGetByReferenceIDAndLanguage(ctx context.Context, referenceID uuid.UUID, language string) (*transcribe.Transcribe, error)
	TranscribeGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcribe.Transcribe, error)
	// TranscribeGetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*transcribe.Transcribe, error)
	TranscribeSetStatus(ctx context.Context, id uuid.UUID, status transcribe.Status) error

	// transcript
	TranscriptCreate(ctx context.Context, t *transcript.Transcript) error
	TranscriptGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error)
	TranscriptGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcript.Transcript, error)
	TranscriptDelete(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// List of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000"
)

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
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
