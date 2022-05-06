package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// activeflow
	ActiveflowCreate(ctx context.Context, af *activeflow.Activeflow) error
	ActiveflowGet(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	ActiveflowUpdate(ctx context.Context, af *activeflow.Activeflow) error
	ActiveflowDelete(ctx context.Context, id uuid.UUID) error
	ActiveflowGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*activeflow.Activeflow, error)

	// flow
	FlowCreate(ctx context.Context, f *flow.Flow) error
	FlowDelete(ctx context.Context, id uuid.UUID) error
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*flow.Flow, error)
	FlowGetsByType(ctx context.Context, customerID uuid.UUID, flowType flow.Type, token string, limit uint64) ([]*flow.Flow, error)
	FlowSetToCache(ctx context.Context, f *flow.Flow) error
	FlowUpdate(ctx context.Context, id uuid.UUID, name, detail string, actions []action.Action) error
	FlowUpdateActions(ctx context.Context, id uuid.UUID, actions []action.Action) error

	VariableCreate(ctx context.Context, t *variable.Variable) error
	VariableGet(ctx context.Context, id uuid.UUID) (*variable.Variable, error)
	VariableUpdate(ctx context.Context, t *variable.Variable) (*variable.Variable, error)
}

// handler database handler
type handler struct {
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
