package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler DBHandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	BridgeCreate(ctx context.Context, b *bridge.Bridge) error
	BridgeEnd(ctx context.Context, asteriskID, id, timestamp string) error
	BridgeGet(ctx context.Context, asteriskID, id string) (*bridge.Bridge, error)

	CallCreate(ctx context.Context, call *call.Call) error
	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error)
	CallSetAction(ctx context.Context, id uuid.UUID, action *action.Action) error
	CallSetFlowID(ctx context.Context, id, flowID uuid.UUID, tmUpdate string) error
	CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy, tmUpdate string) error
	CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmUpdate string) error

	ChannelCreate(ctx context.Context, channel *channel.Channel) error
	ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup ari.ChannelCause) error
	ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error)
	ChannelSetData(ctx context.Context, asteriskID, id string, data map[string]interface{}) error
	ChannelSetDataAndStasis(ctx context.Context, asteriskID, id string, data map[string]interface{}, stasis string) error
	ChannelSetStasis(ctx context.Context, asteriskID, id, stasis string) error
	ChannelSetState(ctx context.Context, asteriskID, id, timestamp string, state ari.ChannelState) error
}

// handler database handler
type handler struct {
	db *sql.DB
}

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

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
