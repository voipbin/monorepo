package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler DBHandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	CallsGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Call, error)

	ConferenceGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Conference, error)

	UserCreate(ctx context.Context, b *models.User) error
	UserGet(ctx context.Context, id uint64) (*models.User, error)
	UserGetFromDB(ctx context.Context, id uint64) (*models.User, error)
	UserGetByUsername(ctx context.Context, username string) (*models.User, error)
	UserGets(ctx context.Context) ([]*models.User, error)
	UserSetToCache(ctx context.Context, u *models.User) error
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

const defaultDelayTimeout = time.Millisecond * 30

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
