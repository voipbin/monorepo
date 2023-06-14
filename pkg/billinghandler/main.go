package billinghandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package billinghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/accounthandler"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

// BillingHandler define
type BillingHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		accountID uuid.UUID,
		referenceType billing.ReferenceType,
		referenceID uuid.UUID,
		costPerUnit float32,
		tmBillingStart string,
	) (*billing.Billing, error)
	Get(ctx context.Context, id uuid.UUID) (*billing.Billing, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*billing.Billing, error)
	UpdateStatusEnd(ctx context.Context, id uuid.UUID, billingDuration float32, tmBillingEnd string) (*billing.Billing, error)

	BillingStart(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType billing.ReferenceType,
		referenceID uuid.UUID,
		tmBillingStart string,
		source *commonaddress.Address,
		destination *commonaddress.Address,
	) error
	BillingEnd(
		ctx context.Context,
		customerID uuid.UUID,
		referenceType billing.ReferenceType,
		referenceID uuid.UUID,
		tmBillingEnd string,
		source *commonaddress.Address,
		destination *commonaddress.Address,
	) error
}

type billingHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	accountHandler accounthandler.AccountHandler
}

var (
	metricsNamespace = "billing_manager"

	promBillingCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "billing_create_total",
			Help:      "Total number of created billing with reference_type.",
		},
		[]string{"reference_type"},
	)

	promBillingEndTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "billing_end_total",
			Help:      "Total number of ended billing type with reference_type.",
		},
		[]string{"reference_type"},
	)
)

var (
	defaultCostPerUnitReferenceTypeCall float32 = 0.020
	defaultCostPerUnitReferenceTypeSMS  float32 = 0.008
)

// NewBillingHandler create a new BillingHandler
func NewBillingHandler(
	reqHandler requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	accountHandler accounthandler.AccountHandler,
) BillingHandler {
	h := &billingHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            db,
		notifyHandler: notifyHandler,

		accountHandler: accountHandler,
	}

	return h
}
