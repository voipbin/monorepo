package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler -source ./main.go DBHandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {

	// number
	NumberCreate(ctx context.Context, n *models.Number) error
	NumberDelete(ctx context.Context, id uuid.UUID) error
	NumberGet(ctx context.Context, id uuid.UUID) (*models.Number, error)
	NumberGetByNumber(ctx context.Context, numb string) (*models.Number, error)
	NumberGetFromCache(ctx context.Context, id uuid.UUID) (*models.Number, error)
	NumberGetFromCacheByNumber(ctx context.Context, numb string) (*models.Number, error)
	NumberGetFromDB(ctx context.Context, id uuid.UUID) (*models.Number, error)
	NumberGetFromDBByNumber(ctx context.Context, numb string) (*models.Number, error)
	NumberGets(ctx context.Context, userID uint64, size uint64, token string) ([]*models.Number, error)
	NumberGetsByFlowID(ctx context.Context, flowID uuid.UUID, size uint64, token string) ([]*models.Number, error)
	NumberSetToCache(ctx context.Context, num *models.Number) error
	NumberSetToCacheByNumber(ctx context.Context, num *models.Number) error
	NumberUpdate(ctx context.Context, numb *models.Number) error
	NumberUpdateToCache(ctx context.Context, id uuid.UUID) error
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

const defaultDelayTimeout = time.Millisecond * 150

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
