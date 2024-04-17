package accounthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package accounthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

// AccountHandler define
type AccountHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name string, detail string, paymentType account.PaymentType, payemntMethod account.PaymentMethod) (*account.Account, error)
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*account.Account, error)
	SubtractBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error)
	AddBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*account.Account, error)
	UpdatePaymentInfo(ctx context.Context, id uuid.UUID, paymentType account.PaymentType, paymentMethod account.PaymentMethod) (*account.Account, error)

	Delete(ctx context.Context, id uuid.UUID) (*account.Account, error)

	IsValidBalance(ctx context.Context, accountID uuid.UUID, billingType billing.ReferenceType, country string, count int) (bool, error)

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
