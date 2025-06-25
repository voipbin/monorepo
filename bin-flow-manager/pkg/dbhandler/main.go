package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {

	// activeflow
	ActiveflowCreate(ctx context.Context, af *activeflow.Activeflow) error
	ActiveflowGet(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	ActiveflowUpdate(ctx context.Context, id uuid.UUID, fields map[activeflow.Field]any) error
	ActiveflowDelete(ctx context.Context, id uuid.UUID) error
	ActiveflowGets(ctx context.Context, token string, size uint64, filters map[activeflow.Field]any) ([]*activeflow.Activeflow, error)
	ActiveflowGetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	ActiveflowReleaseLock(ctx context.Context, id uuid.UUID) error

	// flow
	FlowCreate(ctx context.Context, f *flow.Flow) error
	FlowDelete(ctx context.Context, id uuid.UUID) error
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGets(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error)
	FlowSetToCache(ctx context.Context, f *flow.Flow) error
	FlowUpdate(ctx context.Context, id uuid.UUID, fields map[flow.Field]any) error

	VariableCreate(ctx context.Context, t *variable.Variable) error
	VariableGet(ctx context.Context, id uuid.UUID) (*variable.Variable, error)
	VariableUpdate(ctx context.Context, t *variable.Variable) error
}

// handler database handler
type handler struct {
	util  utilhandler.UtilHandler
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// list of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:000"
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		util:  utilhandler.NewUtilHandler(),
		db:    db,
		cache: cache,
	}
	return h
}
