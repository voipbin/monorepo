package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// bridges
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

	// calls
	CallAddChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error
	CallAddRecordIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error
	CallApplicationAMDGet(ctx context.Context, channelID string) (*callapplication.AMD, error)
	CallApplicationAMDSet(ctx context.Context, channelID string, app *callapplication.AMD) error
	CallCreate(ctx context.Context, call *call.Call) error
	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error)
	CallGetFromCache(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetFromDB(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*call.Call, error)
	CallRemoveChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error
	CallSetAction(ctx context.Context, id uuid.UUID, action *action.Action) error
	CallSetAsteriskID(ctx context.Context, id uuid.UUID, asteriskID string, tmUpdate string) error
	CallSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error
	CallSetConfbridgeID(ctx context.Context, id, confbridgeID uuid.UUID) error
	CallSetFlowID(ctx context.Context, id, flowID uuid.UUID) error
	CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy, tmUpdate string) error
	CallSetMasterCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) error
	CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmUpdate string) error
	CallSetRecordID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error
	CallSetToCache(ctx context.Context, call *call.Call) error
	CallTXAddChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error
	CallTXFinish(tx *sql.Tx, commit bool)
	CallTXRemoveChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error
	CallTXStart(id uuid.UUID) (*sql.Tx, *call.Call, error)
	CallUpdateToCache(ctx context.Context, id uuid.UUID) error

	// calldtmfs
	CallDTMFGet(ctx context.Context, id uuid.UUID) (string, error)
	CallDTMFReset(ctx context.Context, id uuid.UUID) error
	CallDTMFSet(ctx context.Context, id uuid.UUID, dtmf string) error

	// channels
	ChannelCreate(ctx context.Context, channel *channel.Channel) error
	ChannelEnd(ctx context.Context, id, timestamp string, hangup ari.ChannelCause) error
	ChannelGet(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetFromCache(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetFromDB(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetUntilTimeout(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetUntilTimeoutWithStasis(ctx context.Context, id string) (*channel.Channel, error)
	ChannelIsExist(id string, timeout time.Duration) bool
	ChannelSetBridgeID(ctx context.Context, id, bridgeID string) error
	ChannelSetData(ctx context.Context, id string, data map[string]interface{}) error
	ChannelSetStasisNameAndStasisData(ctx context.Context, id string, stasisName string, stasisData map[string]string) error
	ChannelSetDataItem(ctx context.Context, id string, key string, value interface{}) error
	ChannelSetDirection(ctx context.Context, id string, direction channel.Direction) error
	ChannelSetPlaybackID(ctx context.Context, id string, playbackID string) error
	ChannelSetSIPCallID(ctx context.Context, id string, sipID string) error
	ChannelSetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error
	ChannelSetStasis(ctx context.Context, id, stasis string) error
	ChannelSetState(ctx context.Context, id, timestamp string, state ari.ChannelState) error
	ChannelSetType(ctx context.Context, id string, cType channel.Type) error
	ChannelSetToCache(ctx context.Context, channel *channel.Channel) error
	ChannelUpdateToCache(ctx context.Context, id string) error

	// confbridges
	ConfbridgeAddChannelCallID(ctx context.Context, id uuid.UUID, channelID string, callID uuid.UUID) error
	ConfbridgeAddRecordIDs(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error
	ConfbridgeCreate(ctx context.Context, cb *confbridge.Confbridge) error
	ConfbridgeDelete(ctx context.Context, id uuid.UUID) error
	ConfbridgeGet(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	ConfbridgeGetByBridgeID(ctx context.Context, bridgeID string) (*confbridge.Confbridge, error)
	ConfbridgeGetFromCache(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	ConfbridgeGetFromDB(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	ConfbridgeUpdateToCache(ctx context.Context, id uuid.UUID) error
	ConfbridgeRemoveChannelCallID(ctx context.Context, id uuid.UUID, channelID string) error
	ConfbridgeSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error
	ConfbridgeSetRecordID(ctx context.Context, id uuid.UUID, recordID uuid.UUID) error
	ConfbridgeSetToCache(ctx context.Context, data *confbridge.Confbridge) error

	// external media
	ExternalMediaDelete(ctx context.Context, callID uuid.UUID) error
	ExternalMediaGet(ctx context.Context, callID uuid.UUID) (*externalmedia.ExternalMedia, error)
	ExternalMediaSet(ctx context.Context, callID uuid.UUID, data *externalmedia.ExternalMedia) error

	// recordings
	RecordingCreate(ctx context.Context, c *recording.Recording) error
	RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	RecordingGetByFilename(ctx context.Context, filename string) (*recording.Recording, error)
	RecordingGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*recording.Recording, error)
	RecordingSetStatus(ctx context.Context, id uuid.UUID, status recording.Status, timestamp string) error
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

// list of default values
const (
	defaultDelayTimeout = time.Millisecond * 150
	DefaultTimeStamp    = "9999-01-01 00:00:00.000000" //nolint:varcheck,deadcode // this is fine
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
