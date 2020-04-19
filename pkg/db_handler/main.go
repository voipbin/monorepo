package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler DBHandler

import (
	"context"
	"database/sql"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	ChannelCreate(ctx context.Context, channel channel.Channel) error
	ChannelGet(ctx context.Context, asteriskID, id string) (*channel.Channel, error)
	ChannelEnd(ctx context.Context, asteriskID, id, timestamp string, hangup int) error
}

// Handler database handler
type Handler struct {
	db *sql.DB
}

// NewHandler creates DBHandler
func NewHandler(db *sql.DB) DBHandler {
	handler := &Handler{
		db: db,
	}
	return handler
}
