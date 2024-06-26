package numberhandlertelnyx

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandlertelnyx -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/requestexternal"
)

// NumberHandlerTelnyx is interface for service handle
type NumberHandlerTelnyx interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
	NumberPurchase(num string) (*providernumber.ProviderNumber, error)
	NumberRelease(ctx context.Context, num *number.Number) error
	NumberUpdateTags(ctx context.Context, number *number.Number, tags []string) error
}

// numberHandlerTelnyx structure for service handle
type numberHandlerTelnyx struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler

	connectionID string
	profileID    string
	token        string

	requestExternal requestexternal.RequestExternal
}

var (
	metricsNamespace = "number_manager"

	promCreateTotal = prometheus.NewCounterVec(
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
		promCreateTotal,
	)
}

// NewNumberHandler returns new service handler
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, connectionID string, profileID string, token string) NumberHandlerTelnyx {

	reqExternal := requestexternal.NewRequestExternal()

	h := &numberHandlerTelnyx{
		reqHandler: r,
		db:         db,

		connectionID: connectionID,
		profileID:    profileID,
		token:        token,

		requestExternal: reqExternal,
	}

	return h
}
