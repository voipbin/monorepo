package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// bridges
	BridgeAddChannelID(ctx context.Context, id, channelID string) error
	BridgeCreate(ctx context.Context, b *bridge.Bridge) error
	BridgeEnd(ctx context.Context, id string) error
	BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeRemoveChannelID(ctx context.Context, id, channelID string) error

	// calls
	CallAddChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error
	CallAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	CallApplicationAMDGet(ctx context.Context, channelID string) (*callapplication.AMD, error)
	CallApplicationAMDSet(ctx context.Context, channelID string, app *callapplication.AMD) error
	CallCreate(ctx context.Context, call *call.Call) error
	CallDelete(ctx context.Context, id uuid.UUID) error
	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error)
	CallList(ctx context.Context, size uint64, token string, filters map[call.Field]any) ([]*call.Call, error)
	CallUpdate(ctx context.Context, id uuid.UUID, fields map[call.Field]any) error
	CallRemoveChainedCallID(ctx context.Context, id, chainedCallID uuid.UUID) error
	CallSetActionAndActionNextHold(ctx context.Context, id uuid.UUID, action *fmaction.Action, hold bool) error
	CallSetActionNextHold(ctx context.Context, id uuid.UUID, hold bool) error
	CallSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error
	CallSetChannelIDAndBridgeID(ctx context.Context, id uuid.UUID, channelID string, bridgeID string) error
	CallSetConfbridgeID(ctx context.Context, id, confbridgeID uuid.UUID) error
	CallSetData(ctx context.Context, id uuid.UUID, data map[call.DataType]string) error
	CallSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error
	CallSetFlowID(ctx context.Context, id, flowID uuid.UUID) error
	CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy) error
	CallSetMasterCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) error
	CallSetMuteDirection(ctx context.Context, id uuid.UUID, muteDirection call.MuteDirection) error
	CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status) error
	CallSetStatusProgressing(ctx context.Context, id uuid.UUID) error
	CallSetStatusRinging(ctx context.Context, id uuid.UUID) error
	CallSetRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	CallSetForRouteFailover(ctx context.Context, id uuid.UUID, channelID string, dialrouteID uuid.UUID) error
	CallTXAddChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error
	CallTXFinish(tx *sql.Tx, commit bool)
	CallTXRemoveChainedCallID(tx *sql.Tx, id, chainedCallID uuid.UUID) error
	CallTXStart(id uuid.UUID) (*sql.Tx, *call.Call, error)

	// channels
	ChannelCreate(ctx context.Context, channel *channel.Channel) error
	ChannelEndAndDelete(ctx context.Context, id string, hangup ari.ChannelCause) error
	ChannelGet(ctx context.Context, id string) (*channel.Channel, error)
	ChannelList(ctx context.Context, size uint64, token string, filters map[string]string) ([]*channel.Channel, error)
	ChannelGetsForRecovery(
		ctx context.Context,
		asteriskID string,
		channelType channel.Type,
		startTime string,
		endTime string,
		size uint64,
	) ([]*channel.Channel, error)
	ChannelSetBridgeID(ctx context.Context, id, bridgeID string) error
	ChannelSetData(ctx context.Context, id string, data map[string]interface{}) error
	ChannelSetStasisInfo(ctx context.Context, id string, chType channel.Type, stasisName string, stasisData map[channel.StasisDataType]string, direction channel.Direction) error
	ChannelSetDataItem(ctx context.Context, id string, key string, value interface{}) error
	ChannelSetDirection(ctx context.Context, id string, direction channel.Direction) error
	ChannelSetMuteDirection(ctx context.Context, id string, muteDirection channel.MuteDirection) error
	ChannelSetPlaybackID(ctx context.Context, id string, playbackID string) error
	ChannelSetSIPCallID(ctx context.Context, id string, sipID string) error
	ChannelSetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error
	ChannelSetStasis(ctx context.Context, id, stasis string) error
	ChannelSetStateAnswer(ctx context.Context, id string, state ari.ChannelState) error
	ChannelSetStateRinging(ctx context.Context, id string, state ari.ChannelState) error
	ChannelSetType(ctx context.Context, id string, cType channel.Type) error

	// confbridges
	ConfbridgeAddChannelCallID(ctx context.Context, id uuid.UUID, channelID string, callID uuid.UUID) error
	ConfbridgeAddRecordingIDs(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	ConfbridgeCreate(ctx context.Context, cb *confbridge.Confbridge) error
	ConfbridgeDelete(ctx context.Context, id uuid.UUID) error
	ConfbridgeGet(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	ConfbridgeGetByBridgeID(ctx context.Context, bridgeID string) (*confbridge.Confbridge, error)
	ConfbridgeList(ctx context.Context, size uint64, token string, filters map[confbridge.Field]any) ([]*confbridge.Confbridge, error)
	ConfbridgeUpdate(ctx context.Context, id uuid.UUID, fields map[confbridge.Field]any) error
	ConfbridgeRemoveChannelCallID(ctx context.Context, id uuid.UUID, channelID string) error
	ConfbridgeSetBridgeID(ctx context.Context, id uuid.UUID, bridgeID string) error
	ConfbridgeSetExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) error
	ConfbridgeSetFlags(ctx context.Context, id uuid.UUID, flags []confbridge.Flag) error
	ConfbridgeSetRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) error
	ConfbridgeSetStatus(ctx context.Context, id uuid.UUID, status confbridge.Status) error

	// external media
	ExternalMediaDelete(ctx context.Context, externalMediaID uuid.UUID) error
	ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error)
	ExternalMediaGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*externalmedia.ExternalMedia, error)
	ExternalMediaSet(ctx context.Context, data *externalmedia.ExternalMedia) error

	// groupcall
	GroupcallGet(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)
	GroupcallList(ctx context.Context, size uint64, token string, filters map[groupcall.Field]any) ([]*groupcall.Groupcall, error)
	GroupcallUpdate(ctx context.Context, id uuid.UUID, fields map[groupcall.Field]any) error
	GroupcallCreate(ctx context.Context, data *groupcall.Groupcall) error
	GroupcallDecreaseCallCount(ctx context.Context, id uuid.UUID) error
	GroupcallDecreaseGroupcallCount(ctx context.Context, id uuid.UUID) error
	GroupcallSetAnswerCallID(ctx context.Context, id uuid.UUID, answerCallID uuid.UUID) error
	GroupcallSetAnswerGroupcallID(ctx context.Context, id uuid.UUID, answerGroupcallID uuid.UUID) error
	GroupcallSetStatus(ctx context.Context, id uuid.UUID, status groupcall.Status) error
	GroupcallSetCallIDsAndCallCountAndDialIndex(ctx context.Context, id uuid.UUID, callIDs []uuid.UUID, callCount int, dialIndex int) error
	GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx context.Context, id uuid.UUID, groupcallIDs []uuid.UUID, groupcallCount int, dialIndex int) error
	GroupcallDelete(ctx context.Context, id uuid.UUID) error

	// recordings
	RecordingCreate(ctx context.Context, c *recording.Recording) error
	RecordingDelete(ctx context.Context, id uuid.UUID) error
	RecordingGet(ctx context.Context, id uuid.UUID) (*recording.Recording, error)
	RecordingGetByRecordingName(ctx context.Context, recordingName string) (*recording.Recording, error)
	RecordingList(ctx context.Context, size uint64, token string, filters map[recording.Field]any) ([]*recording.Recording, error)
	RecordingUpdate(ctx context.Context, id uuid.UUID, fields map[recording.Field]any) error
	RecordingSetStatus(ctx context.Context, id uuid.UUID, status recording.Status) error
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

// list of default values
const (
	DefaultTimeStamp = "9999-01-01T00:00:00.000000Z" //nolint:varcheck,deadcode // this is fine
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
