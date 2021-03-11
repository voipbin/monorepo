package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler DBHandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	ActiveFlowCreate(ctx context.Context, af *activeflow.ActiveFlow) error
	ActiveFlowGet(ctx context.Context, id uuid.UUID) (*activeflow.ActiveFlow, error)
	ActiveFlowGetFromCache(ctx context.Context, id uuid.UUID) (*activeflow.ActiveFlow, error)
	ActiveFlowSet(ctx context.Context, af *activeflow.ActiveFlow) error
	ActiveFlowSetToCache(ctx context.Context, flow *activeflow.ActiveFlow) error

	FlowCreate(ctx context.Context, f *flow.Flow) error
	FlowDelete(ctx context.Context, id uuid.UUID) error
	FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetFromCache(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetFromDB(ctx context.Context, id uuid.UUID) (*flow.Flow, error)
	FlowGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*flow.Flow, error)
	FlowSetToCache(ctx context.Context, f *flow.Flow) error
	FlowUpdate(ctx context.Context, f *flow.Flow) error
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

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
