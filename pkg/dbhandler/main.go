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
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferenceconfbridge"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {
	// conferences
	ConferenceAddCallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceAddRecordIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error
	ConferenceCreate(ctx context.Context, cf *conference.Conference) error
	ConferenceEnd(ctx context.Context, id uuid.UUID) error
	ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGetFromCache(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGetFromDB(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*conference.Conference, error)
	ConferenceGetsWithType(ctx context.Context, customerID uuid.UUID, confType conference.Type, size uint64, token string) ([]*conference.Conference, error)
	ConferenceRemoveCallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceSet(ctx context.Context, id uuid.UUID, name, detail string, timeout int, webhookURI string, preActions, postActions []fmaction.Action) error
	ConferenceSetData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error
	ConferenceSetRecordID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error
	ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error
	ConferenceSetToCache(ctx context.Context, conference *conference.Conference) error
	ConferenceUpdateToCache(ctx context.Context, id uuid.UUID) error

	ConferenceConfbridgeSet(ctx context.Context, data *conferenceconfbridge.ConferenceConfbridge) error
	ConferenceConfbridgeGet(ctx context.Context, confbridgeID uuid.UUID) (*conferenceconfbridge.ConferenceConfbridge, error)
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000"
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
