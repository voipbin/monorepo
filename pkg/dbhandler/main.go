package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler DBHandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	BridgeAddChannelID(ctx context.Context, id, channelID string) error
	BridgeCreate(ctx context.Context, b *bridge.Bridge) error
	BridgeEnd(ctx context.Context, id, timestamp string) error
	BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeGetFromCache(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeGetFromDB(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeGetUntilTimeout(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeIsExist(id string, timeout time.Duration) bool
	BridgeRemoveChannelID(ctx context.Context, id, channelID string) error
	BridgeSetToCache(ctx context.Context, bridge *bridge.Bridge) error
	BridgeUpdateToCache(ctx context.Context, id string) error

	CallCreate(ctx context.Context, call *call.Call) error
	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetFromCache(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetFromDB(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error)
	CallSetAction(ctx context.Context, id uuid.UUID, action *action.Action) error
	CallSetConferenceID(ctx context.Context, id, conferenceID uuid.UUID) error
	CallSetFlowID(ctx context.Context, id, flowID uuid.UUID) error
	CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy, tmUpdate string) error
	CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmUpdate string) error
	CallSetToCache(ctx context.Context, call *call.Call) error
	CallUpdateToCache(ctx context.Context, id uuid.UUID) error

	ChannelCreate(ctx context.Context, channel *channel.Channel) error
	ChannelEnd(ctx context.Context, id, timestamp string, hangup ari.ChannelCause) error
	ChannelGet(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetFromCache(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetFromDB(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetUntilTimeout(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetUntilTimeoutWithStasis(ctx context.Context, id string) (*channel.Channel, error)
	ChannelSetBridgeID(ctx context.Context, id, bridgeID string) error
	ChannelSetData(ctx context.Context, id string, data map[string]interface{}) error
	ChannelSetDataAndStasis(ctx context.Context, id string, data map[string]interface{}, stasis string) error
	ChannelIsExist(id string, timeout time.Duration) bool
	ChannelSetStasis(ctx context.Context, id, stasis string) error
	ChannelSetState(ctx context.Context, id, timestamp string, state ari.ChannelState) error
	ChannelSetToCache(ctx context.Context, channel *channel.Channel) error
	ChannelSetTransport(ctx context.Context, id string, transport channel.Transport) error
	ChannelUpdateToCache(ctx context.Context, id string) error

	ConferenceAddCallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceCreate(ctx context.Context, cf *conference.Conference) error
	ConferenceEnd(ctx context.Context, id uuid.UUID) error
	ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGetFromCache(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceGetFromDB(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceRemoveCallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error
	ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error
	ConferenceSetToCache(ctx context.Context, conference *conference.Conference) error
	ConferenceUpdateToCache(ctx context.Context, id uuid.UUID) error
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

const defaultDelayTimeout = time.Millisecond * 30

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
