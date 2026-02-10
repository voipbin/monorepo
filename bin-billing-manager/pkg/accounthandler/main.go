package accounthandler

//go:generate mockgen -package accounthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// AccountHandler define
type AccountHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name string, detail string, paymentType account.PaymentType, payemntMethod account.PaymentMethod) (*account.Account, error)
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error)
	List(ctx context.Context, size uint64, token string, filters map[account.Field]any) ([]*account.Account, error)
	SubtractBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error)
	SubtractBalanceWithCheck(ctx context.Context, accountID uuid.UUID, amount float32) (*account.Account, error)
	AddBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*account.Account, error)
	UpdatePaymentInfo(ctx context.Context, id uuid.UUID, paymentType account.PaymentType, paymentMethod account.PaymentMethod) (*account.Account, error)
	UpdatePlanType(ctx context.Context, id uuid.UUID, planType account.PlanType) (*account.Account, error)

	Delete(ctx context.Context, id uuid.UUID) (*account.Account, error)

	IsValidBalance(ctx context.Context, accountID uuid.UUID, billingType billing.ReferenceType, country string, count int) (bool, error)
	IsValidBalanceByCustomerID(ctx context.Context, customerID uuid.UUID, billingType billing.ReferenceType, country string, count int) (bool, error)
	IsValidResourceLimit(ctx context.Context, accountID uuid.UUID, resourceType account.ResourceType) (bool, error)
	IsValidResourceLimitByCustomerID(ctx context.Context, customerID uuid.UUID, resourceType account.ResourceType) (bool, error)

	EventCUCustomerCreated(ctx context.Context, cu *cucustomer.Customer) error
	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
}

// accountHandler define
type accountHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

var (
	metricsNamespace = "billing_manager"

	promAccountCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "account_create_total",
			Help:      "Total number of created account.",
		},
	)
)

// NewAccountHandler returns a new AccountHandler
func NewAccountHandler(reqHandler requesthandler.RequestHandler, db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) AccountHandler {
	return &accountHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,
	}
}
