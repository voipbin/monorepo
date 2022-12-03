package numberhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/util"
)

// NumberHandler is interface for service handle
type NumberHandler interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)

	Create(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*number.Number, error)
	Get(ctx context.Context, id uuid.UUID) (*number.Number, error)
	GetByNumber(ctx context.Context, num string) (*number.Number, error)
	GetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*number.Number, error)

	Delete(ctx context.Context, id uuid.UUID) (*number.Number, error)

	RemoveNumbersFlowID(ctx context.Context, flowID uuid.UUID) error

	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*number.Number, error)
	UpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*number.Number, error)
}

// numberHandler structure for service handle
type numberHandler struct {
	util          util.Util
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	numberHandlerTelnyx numberhandlertelnyx.NumberHandlerTelnyx
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

	h := &numberHandler{
		util:          util.NewUtil(),
		reqHandler:    r,
		db:            db,
		notifyHandler: notifyHandler,

		numberHandlerTelnyx: nHandlerTelnyx,
	}

	return h
}
