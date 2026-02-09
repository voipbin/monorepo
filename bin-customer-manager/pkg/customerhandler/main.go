package customerhandler

//go:generate mockgen -package customerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonbilling "monorepo/bin-common-handler/models/billing"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/cachehandler"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

// CustomerHandler interface
type CustomerHandler interface {
	Create(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
	) (*customer.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	Get(ctx context.Context, id uuid.UUID) (*customer.Customer, error)
	List(ctx context.Context, size uint64, token string, filters map[customer.Field]any) ([]*customer.Customer, error)
	UpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
	) (*customer.Customer, error)
	UpdateBillingAccountID(ctx context.Context, id uuid.UUID, billingAccountID uuid.UUID) (*customer.Customer, error)

	Signup(
		ctx context.Context,
		name string,
		detail string,
		email string,
		phoneNumber string,
		address string,
		webhookMethod customer.WebhookMethod,
		webhookURI string,
	) (*customer.Customer, error)
	EmailVerify(ctx context.Context, token string) (*customer.Customer, error)

	RunCleanupUnverified(ctx context.Context)

	IsValidBalance(ctx context.Context, customerID uuid.UUID, billingType bmbilling.ReferenceType, country string, count int) (bool, error)
	IsValidResourceLimit(ctx context.Context, customerID uuid.UUID, resourceType commonbilling.ResourceType) (bool, error)
}

type customerHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewCustomerHandler return UserHandler interface
func NewCustomerHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, cache cachehandler.CacheHandler, notifyHandler notifyhandler.NotifyHandler) CustomerHandler {
	return &customerHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		cache:         cache,
		notifyHandler: notifyHandler,
	}
}
