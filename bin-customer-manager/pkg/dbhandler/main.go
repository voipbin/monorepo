package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	AccesskeyCreate(ctx context.Context, c *accesskey.Accesskey) error
	AccesskeyDelete(ctx context.Context, id uuid.UUID) error
	AccesskeyGet(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error)
	AccesskeyList(ctx context.Context, size uint64, token string, filters map[accesskey.Field]any) ([]*accesskey.Accesskey, error)
	AccesskeyUpdate(ctx context.Context, id uuid.UUID, fields map[accesskey.Field]any) error

	CustomerCreate(ctx context.Context, b *customer.Customer) error
	CustomerDelete(ctx context.Context, id uuid.UUID) error
	CustomerGet(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	CustomerList(ctx context.Context, size uint64, token string, filters map[customer.Field]any) ([]*customer.Customer, error)
	CustomerUpdate(ctx context.Context, id uuid.UUID, fields map[customer.Field]any) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)


// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
