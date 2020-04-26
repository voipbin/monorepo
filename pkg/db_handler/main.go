package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler DBHandler

import (
	"context"
	"database/sql"
	"errors"

	uuid "github.com/satori/go.uuid"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	ChannelCreate(ctx context.Context, channel *channel.Channel) error
	ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup ari.ChannelCause) error
	ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error)
	ChannelSetData(ctx context.Context, asteriskID, id, timestamp string, data map[string]interface{}) error
	ChannelSetState(ctx context.Context, asteriskID, id, timestamp string, state ari.ChannelState) error

	CallCreate(ctx context.Context, call *call.Call) error
	CallGet(ctx context.Context, id uuid.UUID) (*call.Call, error)
	CallGetByChannelID(ctx context.Context, channelID string) (*call.Call, error)
	CallSetHangup(ctx context.Context, id uuid.UUID, reason call.HangupReason, hangupBy call.HangupBy, tmUpdate string) error
	CallSetStatus(ctx context.Context, id uuid.UUID, status call.Status, tmUpdate string) error
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
