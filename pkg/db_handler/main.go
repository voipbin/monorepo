package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/flow-manager/pkg/db_handler DBHandler

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	flow "gitlab.com/voipbin/bin-manager/flow-manager/pkg/flow"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	FlowCreate(ctx context.Context, flow *flow.Flow) error
	FlowGet(ctx context.Context, id, revision uuid.UUID) (*flow.Flow, error)
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
