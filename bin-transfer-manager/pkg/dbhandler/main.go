package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-transfer-manager/models/transfer"
	"monorepo/bin-transfer-manager/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {
	TransferCreate(ctx context.Context, tr *transfer.Transfer) error
	TransferGet(ctx context.Context, id uuid.UUID) (*transfer.Transfer, error)
	TransferGetByTransfererCallID(ctx context.Context, transfererCallID uuid.UUID) (*transfer.Transfer, error)
	TransferGetByGroupcallID(ctx context.Context, groupcallID uuid.UUID) (*transfer.Transfer, error)
	TransferUpdate(ctx context.Context, tr *transfer.Transfer) error
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
