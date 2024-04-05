package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/streaming"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
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
