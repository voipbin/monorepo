package accounthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package accounthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

// AccountHandler define
type AccountHandler interface {
	Create(ctx context.Context, customerID uuid.UUID) (*account.Account, error)
	Get(ctx context.Context, id uuid.UUID) (*account.Account, error)
	Gets(ctx context.Context, size uint64, token string) ([]*account.Account, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*account.Account, error)
	SubtractBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error)
	AddBalance(ctx context.Context, accountID uuid.UUID, balance float32) (*account.Account, error)

	IsValidBalanceByCustomerID(ctx context.Context, customerID uuid.UUID) (bool, error)
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
