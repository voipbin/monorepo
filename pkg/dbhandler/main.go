package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {
	// conferences
	ConferenceAddConferencecallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	ConferenceCreate(ctx context.Context, cf *conference.Conference) error
	ConferenceEnd(ctx context.Context, id uuid.UUID) error
	ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGetByConfbridgeID(ctx context.Context, confbridgeID uuid.UUID) (*conference.Conference, error)
	ConferenceGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*conference.Conference, error)
	ConferenceGetsWithType(ctx context.Context, customerID uuid.UUID, confType conference.Type, size uint64, token string) ([]*conference.Conference, error)
	ConferenceRemoveConferencecallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceSet(ctx context.Context, id uuid.UUID, name, detail string, timeout int, preActions, postActions []fmaction.Action) error
	ConferenceSetData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error
	ConferenceSetRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error

	// conferencecalls
	ConferencecallCreate(ctx context.Context, cf *conferencecall.Conferencecall) error
	ConferencecallGet(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error)
	ConferencecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error)
	ConferencecallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*conferencecall.Conferencecall, error)
	ConferencecallGetsByConferenceID(ctx context.Context, conferenceID uuid.UUID, size uint64, token string) ([]*conferencecall.Conferencecall, error)
	ConferencecallUpdateStatus(ctx context.Context, id uuid.UUID, status conferencecall.Status) error
	ConferencecallDelete(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// list of default variables
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000"
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
