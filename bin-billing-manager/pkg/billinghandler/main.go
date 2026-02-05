package billinghandler

//go:generate mockgen -package billinghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	mmmessage "monorepo/bin-message-manager/models/message"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/dbhandler"
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
		tmBillingStart *time.Time,
	) (*billing.Billing, error)
	Get(ctx context.Context, id uuid.UUID) (*billing.Billing, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error)
	List(ctx context.Context, size uint64, token string, filters map[billing.Field]any) ([]*billing.Billing, error)
	UpdateStatusEnd(ctx context.Context, id uuid.UUID, billingDuration float32, tmBillingEnd *time.Time) (*billing.Billing, error)

	EventCMCallProgressing(ctx context.Context, c *cmcall.Call) error
	EventCMCallHangup(ctx context.Context, c *cmcall.Call) error
	EventMMMessageCreated(ctx context.Context, m *mmmessage.Message) error
	EventNMNumberCreated(ctx context.Context, n *nmnumber.Number) error
	EventNMNumberRenewed(ctx context.Context, n *nmnumber.Number) error
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
