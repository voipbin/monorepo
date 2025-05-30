package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {
	Close()

	// number
	NumberCreate(ctx context.Context, n *number.Number) error
	NumberDelete(ctx context.Context, id uuid.UUID) error
	NumberGet(ctx context.Context, id uuid.UUID) (*number.Number, error)

	NumberGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*number.Number, error)
	NumberGetsByTMRenew(ctx context.Context, tmRenew string, size uint64, filters map[string]string) ([]*number.Number, error)

	NumberUpdateInfo(ctx context.Context, id uuid.UUID, callflowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) error
	NumberUpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) error
	NumberUpdateCallFlowID(ctx context.Context, id, flowID uuid.UUID) error
	NumberUpdateMessageFlowID(ctx context.Context, id, flowID uuid.UUID) error
	NumberUpdateTMRenew(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// List of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000"
)

// handler errors
var (
	ErrNotFound = errors.New("record not found")
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
