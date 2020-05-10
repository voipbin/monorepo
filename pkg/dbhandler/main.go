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
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	BridgeAddChannelID(ctx context.Context, id, channelID string) error
	BridgeCreate(ctx context.Context, b *bridge.Bridge) error
	BridgeEnd(ctx context.Context, id, timestamp string) error
	BridgeGet(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeGetUntilTimeout(ctx context.Context, id string) (*bridge.Bridge, error)
	BridgeIsExist(id string, timeout time.Duration) bool
	BridgeRemoveChannelID(ctx context.Context, id, channelID string) error

	CallCreate(ctx context.Context, call *call.Call) error
	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetByChannelIDAndAsteriskID(ctx context.Context, channelID, asteriskID string) (*call.Call, error)
	CallSetAction(ctx context.Context, id uuid.UUID, action *action.Action) error
	CallSetFlowID(ctx context.Context, id, flowID uuid.UUID, tmUpdate string) error
	CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy, tmUpdate string) error
	CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmUpdate string) error

	ChannelCreate(ctx context.Context, channel *channel.Channel) error
	ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup ari.ChannelCause) error
	ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error)
	ChannelGetByID(ctx context.Context, id string) (*channel.Channel, error)
	ChannelGetUntilTimeout(ctx context.Context, asteriskID, id string) (*channel.Channel, error)
	ChannelGetUntilTimeoutWithStasis(ctx context.Context, asteriskID, id string) (*channel.Channel, error)
	ChannelSetBridgeID(ctx context.Context, asteriskID, id, bridgeID string) error
	ChannelSetData(ctx context.Context, asteriskID, id string, data map[string]interface{}) error
	ChannelSetDataAndStasis(ctx context.Context, asteriskID, id string, data map[string]interface{}, stasis string) error
	ChannelIsExist(id, asteriskID string, timeout time.Duration) bool
	ChannelSetStasis(ctx context.Context, asteriskID, id, stasis string) error
	ChannelSetState(ctx context.Context, asteriskID, id, timestamp string, state ari.ChannelState) error

	ConferenceAddCallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceCreate(ctx context.Context, cf *conference.Conference) error
	ConferenceEnd(ctx context.Context, id uuid.UUID) error
	ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error)
	ConferenceRemoveCallID(ctx context.Context, id, callID uuid.UUID) error
	ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error
}

// handler database handler
type handler struct {
	db *sql.DB
}

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

const defaultDelayTimeout = time.Millisecond * 30

// NewHandler creates DBHandler
func NewHandler(db *sql.DB) DBHandler {
	h := &handler{
		db: db,
	}
	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	date := time.Date(2018, 01, 12, 22, 51, 48, 324359102, time.UTC)

	res := date.String()
	res = strings.TrimSuffix(res, " +0000 UTC")

	return res
}
