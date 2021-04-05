package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {

	// number
	NumberCreate(ctx context.Context, n *number.Number) error
	NumberDelete(ctx context.Context, id uuid.UUID) error
	NumberGet(ctx context.Context, id uuid.UUID) (*number.Number, error)
	NumberGetByNumber(ctx context.Context, numb string) (*number.Number, error)
	NumberGetFromCache(ctx context.Context, id uuid.UUID) (*number.Number, error)
	NumberGetFromCacheByNumber(ctx context.Context, numb string) (*number.Number, error)
	NumberGetFromDB(ctx context.Context, id uuid.UUID) (*number.Number, error)
	NumberGetFromDBByNumber(ctx context.Context, numb string) (*number.Number, error)
	NumberGets(ctx context.Context, userID uint64, size uint64, token string) ([]*number.Number, error)
	NumberGetsByFlowID(ctx context.Context, flowID uuid.UUID, size uint64, token string) ([]*number.Number, error)
	NumberSetToCache(ctx context.Context, num *number.Number) error
	NumberSetToCacheByNumber(ctx context.Context, num *number.Number) error
	NumberUpdate(ctx context.Context, numb *number.Number) error
	NumberUpdateToCache(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// List of default values
const (
	defaultDelayTimeout = time.Millisecond * 150
	defaultTimeStamp    = "9999-01-01 00:00:00.000000"
)

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

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
