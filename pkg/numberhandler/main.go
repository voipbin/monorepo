package numberhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandler -destination ./mock_numberhandler.go -source main.go -build_flags=-mod=mod

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
)

// NumberHandler is interface for service handle
type NumberHandler interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)

	CreateNumber(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*number.Number, error)
	GetNumber(ctx context.Context, id uuid.UUID) (*number.Number, error)
	GetNumberByNumber(ctx context.Context, num string) (*number.Number, error)
	GetNumbers(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*number.Number, error)

	ReleaseNumber(ctx context.Context, id uuid.UUID) (*number.Number, error)

	RemoveNumbersFlowID(ctx context.Context, flowID uuid.UUID) error

	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*number.Number, error)
	UpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*number.Number, error)
}

// numberHandler structure for service handle
type numberHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	numHandlerTelnyx numberhandlertelnyx.NumberHandlerTelnyx
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
		reqHandler:    r,
		db:            db,
		notifyHandler: notifyHandler,

		numHandlerTelnyx: nHandlerTelnyx,
	}

	return h
}
