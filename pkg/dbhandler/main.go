package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

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

// DBHandler interface for database handle
type DBHandler interface {

	// number
	NumberCreate(ctx context.Context, n *number.Number) error
	NumberDelete(ctx context.Context, id uuid.UUID) error
	NumberGet(ctx context.Context, id uuid.UUID) (*number.Number, error)
	NumberGetByNumber(ctx context.Context, numb string) (*number.Number, error)
	NumberGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*number.Number, error)
	NumberGetsByFlowID(ctx context.Context, flowID uuid.UUID, size uint64, token string) ([]*number.Number, error)
	NumberUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	NumberUpdateFlowID(ctx context.Context, id, flowID uuid.UUID) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// List of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000"
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

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
