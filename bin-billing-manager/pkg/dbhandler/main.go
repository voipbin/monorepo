package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/cachehandler"
)

// DBHandler interface for billing_manager database handle
type DBHandler interface {
	AccountCreate(ctx context.Context, c *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountList(ctx context.Context, size uint64, token string, filters map[account.Field]any) ([]*account.Account, error)
	AccountListByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*account.Account, error)
	AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error
	AccountAddBalance(ctx context.Context, accountID uuid.UUID, balance float32) error
	AccountSubtractBalance(ctx context.Context, accountID uuid.UUID, balance float32) error
	AccountDelete(ctx context.Context, id uuid.UUID) error

	BillingCreate(ctx context.Context, c *billing.Billing) error
	BillingGet(ctx context.Context, id uuid.UUID) (*billing.Billing, error)
	BillingGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error)
	BillingList(ctx context.Context, size uint64, token string, filters map[billing.Field]any) ([]*billing.Billing, error)
	BillingUpdate(ctx context.Context, id uuid.UUID, fields map[billing.Field]any) error
	BillingSetStatusEnd(ctx context.Context, id uuid.UUID, billingDuration float32, timestamp string) error
	BillingSetStatus(ctx context.Context, id uuid.UUID, status billing.Status) error
	BillingDelete(ctx context.Context, id uuid.UUID) error
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

// list of default values
const (
	DefaultTimeStamp = "9999-01-01T00:00:00.000000Z" //nolint:varcheck,deadcode // this is fine
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
