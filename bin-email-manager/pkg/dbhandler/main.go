package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	EmailCreate(ctx context.Context, e *email.Email) error
	EmailGet(ctx context.Context, id uuid.UUID) (*email.Email, error)
	EmailList(ctx context.Context, token string, size uint64, filters map[email.Field]any) ([]*email.Email, error)
	EmailDelete(ctx context.Context, id uuid.UUID) error
	EmailUpdate(ctx context.Context, id uuid.UUID, fields map[email.Field]any) error
	EmailUpdateStatus(ctx context.Context, id uuid.UUID, status email.Status) error
	EmailUpdateProviderReferenceID(ctx context.Context, id uuid.UUID, providerReferenceID string) error
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
