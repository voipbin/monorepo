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

	NumberList(ctx context.Context, size uint64, token string, filters map[number.Field]any) ([]*number.Number, error)
	NumberGetsByTMRenew(ctx context.Context, tmRenew string, size uint64, filters map[number.Field]any) ([]*number.Number, error)

	NumberUpdate(ctx context.Context, id uuid.UUID, fields map[number.Field]any) error

	NumberGetExistingNumbers(ctx context.Context, numbers []string) ([]string, error)
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

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
