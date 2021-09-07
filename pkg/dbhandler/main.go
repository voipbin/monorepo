package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {

	// transcribe
	TranscribeGetFromCache(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	TranscribeUpdateToCache(ctx context.Context, id uuid.UUID) error
	TranscribeSetToCache(ctx context.Context, t *transcribe.Transcribe) error

	TranscribeAddTranscript(ctx context.Context, id uuid.UUID, t *transcribe.Transcript) error
	TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error
	TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
	TranscribeGetFromDB(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// List of default values
const (
	defaultDelayTimeout = time.Millisecond * 150
	defaultTimeStamp    = "9999-01-01 00:00:00.000000"
)

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
