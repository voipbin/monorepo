package numberhandler

//go:generate mockgen -package numberhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/numberhandlertwilio"
)

// NumberHandler is interface for service handle
type NumberHandler interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)

	Create(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*number.Number, error)
	Get(ctx context.Context, id uuid.UUID) (*number.Number, error)
	List(ctx context.Context, pageSize uint64, pageToken string, filters map[number.Field]any) ([]*number.Number, error)
	Delete(ctx context.Context, id uuid.UUID) (*number.Number, error)
	Register(
		ctx context.Context,
		customerID uuid.UUID,
		num string,
		callFlowID uuid.UUID,
		messageFlowID uuid.UUID,
		name string,
		detail string,
		providerName number.ProviderName,
		providerReferenceID string,
		status number.Status,
		t38Enabled bool,
		emergencyEnabled bool,
	) (*number.Number, error)

	Update(ctx context.Context, id uuid.UUID, fields map[number.Field]any) (*number.Number, error)

	RenewNumbers(ctx context.Context, days int, hours int, tmRenew string) ([]*number.Number, error)

	EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
	EventFlowDeleted(ctx context.Context, f *fmflow.Flow) error
}

// numberHandler structure for service handle
type numberHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	numberHandlerTelnyx numberhandlertelnyx.NumberHandlerTelnyx
	numberHandlerTwilio numberhandlertwilio.NumberHandlerTwilio
}

var (
	metricsNamespace = "number_manager"

	promNumberCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "number_create_total",
			Help:      "Total number of created number type.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promNumberCreateTotal,
	)
}

// NewNumberHandler returns new service handler
func NewNumberHandler(
	r requesthandler.RequestHandler,
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	nHandlerTelnyx numberhandlertelnyx.NumberHandlerTelnyx,
	nHandlerTwilio numberhandlertwilio.NumberHandlerTwilio,
) NumberHandler {

	h := &numberHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    r,
		db:            db,
		notifyHandler: notifyHandler,

		numberHandlerTelnyx: nHandlerTelnyx,
		numberHandlerTwilio: nHandlerTwilio,
	}

	return h
}
