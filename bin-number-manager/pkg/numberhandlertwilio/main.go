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

//nolint:unused,varcheck,deadcode	// reserved
const (
	twilioSID   string = "AC3300cb9426b78c9ce48db86a755166f0"
	twilioToken string = "58c603e14220f52553be7769b209f423"
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
