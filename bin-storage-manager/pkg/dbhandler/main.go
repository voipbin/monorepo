package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/cachehandler"

	"github.com/gofrs/uuid"
)

// DBHandler interface for storage_manager database handle
type DBHandler interface {
	AccountCreate(ctx context.Context, a *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountGets(ctx context.Context, token string, size uint64, filters map[account.Field]any) ([]*account.Account, error)
	AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error
	AccountIncreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error
	AccountDecreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error
	AccountDelete(ctx context.Context, id uuid.UUID) error

	FileCreate(ctx context.Context, f *file.File) error
	FileGet(ctx context.Context, id uuid.UUID) (*file.File, error)
	FileGets(ctx context.Context, token string, size uint64, filters map[file.Field]any) ([]*file.File, error)
	FileUpdate(ctx context.Context, id uuid.UUID, fields map[file.Field]any) error
	FileDelete(ctx context.Context, id uuid.UUID) error
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

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		util:  utilhandler.NewUtilHandler(),
		db:    db,
		cache: cache,
	}
	return h
}
