package numberhandlertwilio

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandlertwilio -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/requestexternal"
)

// NumberHandlerTwilio is interface for service handle
type NumberHandlerTwilio interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
	CreateNumber(customerID uuid.UUID, num string, flowID uuid.UUID, name, detail string) (*number.Number, error)
	ReleaseNumber(ctx context.Context, num *number.Number) error
}

// numberHandlerTwilio structure for service handle
type numberHandlerTwilio struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler

	sid   string
	token string

	requestExternal requestexternal.RequestExternal
}

var (
	metricsNamespace = "number_manager"

	promCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "twilio_number_create_total",
			Help:      "Total number of created number type by twilio.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promCreateTotal,
	)
}

// NewNumberHandler returns new service handler
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, sid string, token string) NumberHandlerTwilio {

	reqExternal := requestexternal.NewRequestExternal()

	h := &numberHandlerTwilio{
		reqHandler: r,
		db:         db,

		sid:   sid,
		token: token,

		requestExternal: reqExternal,
	}

	return h
}
