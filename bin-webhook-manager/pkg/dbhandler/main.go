package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// DBHandler interface for webhook_manager database handle
type DBHandler interface {
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountSet(ctx context.Context, u *account.Account) error
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
