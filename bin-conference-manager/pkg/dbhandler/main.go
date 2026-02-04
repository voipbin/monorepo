package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {
	// conferences
	ConferenceAddConferencecallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	ConferenceAddTranscribeIDs(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error
	ConferenceCreate(ctx context.Context, cf *conference.Conference) error
	ConferenceDelete(ctx context.Context, id uuid.UUID) error
	ConferenceEnd(ctx context.Context, id uuid.UUID) error
	ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGetByConfbridgeID(ctx context.Context, confbridgeID uuid.UUID) (*conference.Conference, error)
	ConferenceList(ctx context.Context, size uint64, token string, filters map[conference.Field]any) ([]*conference.Conference, error)
	ConferenceRemoveConferencecallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceUpdate(ctx context.Context, id uuid.UUID, fields map[conference.Field]any) error

	// conferencecalls
	ConferencecallCreate(ctx context.Context, cf *conferencecall.Conferencecall) error
	ConferencecallGet(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
	ConferencecallList(ctx context.Context, size uint64, token string, filters map[conferencecall.Field]any) ([]*conferencecall.Conferencecall, error)
	ConferencecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error)
	ConferencecallUpdate(ctx context.Context, id uuid.UUID, fields map[conferencecall.Field]any) error
	ConferencecallDelete(ctx context.Context, id uuid.UUID) error
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

// list of default variables
const (
	DefaultTimeStamp = "9999-01-01T00:00:00.000000Z"
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
