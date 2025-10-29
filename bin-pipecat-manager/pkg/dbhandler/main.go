package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"monorepo/bin-pipecat-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	PipecatcallCreate(ctx context.Context, pc *pipecatcall.Pipecatcall) error
	PipecatcallGet(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error)
	PipecatcallUpdate(ctx context.Context, id uuid.UUID, fields map[pipecatcall.Field]any) error
	PipecatcallDelete(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// list of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" //nolint:varcheck,deadcode // this is fine
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
