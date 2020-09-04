package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/flow-manager/pkg/dbhandler DBHandler

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler/models/flow"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	FlowCreate(ctx context.Context, flow *flow.Flow) error
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetFromCache(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowSetToCache(ctx context.Context, flow *flow.Flow) error
	FlowUpdateToCache(ctx context.Context, id uuid.UUID) error
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

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}
