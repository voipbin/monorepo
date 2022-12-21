package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {

	// transcribe
	TranscribeAddTranscript(ctx context.Context, id uuid.UUID, t *transcript.Transcript) error
	TranscribeCreate(ctx context.Context, t *transcribe.Transcribe) error
	TranscribeDelete(ctx context.Context, id uuid.UUID) error
	TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error)
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

// // GetCurTime return current utc time string
// func GetCurTime() string {
// 	now := time.Now().UTC().String()
// 	res := strings.TrimSuffix(now, " +0000 UTC")

// 	return res
// }
