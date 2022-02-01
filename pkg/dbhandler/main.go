package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	CustomerCreate(ctx context.Context, b *customer.Customer) error
	CustomerDelete(ctx context.Context, id uuid.UUID) error
	CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	CustomerGetByUsername(ctx context.Context, username string) (*customer.Customer, error)
	CustomerGets(ctx context.Context, size uint64, token string) ([]*customer.Customer, error)
	CustomerSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, webhookMethod customer.WebhookMethod, webhookURI string) error
	CustomerSetPermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) error
	CustomerSetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
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

// List of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
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
