package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	UserCreate(ctx context.Context, b *user.User) error
	UserGet(ctx context.Context, id uint64) (*user.User, error)
	UserGetFromDB(ctx context.Context, id uint64) (*user.User, error)
	UserGetByUsername(ctx context.Context, username string) (*user.User, error)
	UserGets(ctx context.Context) ([]*user.User, error)
	UserSetToCache(ctx context.Context, u *user.User) error
	UserUpdateToCache(ctx context.Context, id uint64) error
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
