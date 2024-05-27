package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

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

// DBHandler interface for call_manager database handle
type DBHandler interface {
	AccountCreate(ctx context.Context, f *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*account.Account, error)
	AccountIncreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error
	AccountDecreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error
	AccountDelete(ctx context.Context, id uuid.UUID) error

	FileCreate(ctx context.Context, f *file.File) error
	FileGet(ctx context.Context, id uuid.UUID) (*file.File, error)
	FileGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error)
	FileUpdate(ctx context.Context, id uuid.UUID, name, detail string) error
	FileDelete(ctx context.Context, id uuid.UUID) error
	FileUpdateDownloadInfo(ctx context.Context, id uuid.UUID, uriDownload string, tmDownloadExpire string) error
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
