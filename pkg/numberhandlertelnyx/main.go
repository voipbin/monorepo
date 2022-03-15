package numberhandlertelnyx

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandlertelnyx -destination ./mock_numberhandlertelnyx.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal"
)

// NumberHandlerTelnyx is interface for service handle
type NumberHandlerTelnyx interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
	CreateNumber(customerID uuid.UUID, num string, flowID uuid.UUID, name, detail string) (*number.Number, error)
	ReleaseNumber(ctx context.Context, num *number.Number) (*number.Number, error)
}

// numberHandlerTelnyx structure for service handle
type numberHandlerTelnyx struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler

	requestExternal requestexternal.RequestExternal
}

var (
	metricsNamespace = "number_manager"

	promTelnyxCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "telnyx_number_create_total",
			Help:      "Total number of created number type by telnyx.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promTelnyxCreateTotal,
	)
}

// NewNumberHandler returns new service handler
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler) NumberHandlerTelnyx {

	reqExternal := requestexternal.NewRequestExternal()

	h := &numberHandlerTelnyx{
		reqHandler:      r,
		db:              db,
		requestExternal: reqExternal,
	}

	return h
}
