package numberhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

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
	Gets(ctx context.Context, pageSize uint64, pageToken string, filters map[string]string) ([]*number.Number, error)
	Delete(ctx context.Context, id uuid.UUID) (*number.Number, error)

	UpdateInfo(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*number.Number, error)
	UpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*number.Number, error)

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
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) NumberHandler {

	nHandlerTelnyx := numberhandlertelnyx.NewNumberHandler(r, db)
	nHandlerTwilio := numberhandlertwilio.NewNumberHandler(r, db)

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
