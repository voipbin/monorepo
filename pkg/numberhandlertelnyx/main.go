package numberhandlertelnyx

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandlertelnyx -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/providernumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal"
)

// telnyx
const (
	defaultToken              string = "KEY017B6ED1E90D8FC5DB6ED95F1ACFE4F5_WzTaTxsXJCdwOviG4t1xMM"
	defaultConnectionID       string = "2054833017033065613"
	defaultMessagingProfileID string = "40017f8e-49bd-4f16-9e3d-ef103f916228"
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
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler) NumberHandlerTelnyx {

	reqExternal := requestexternal.NewRequestExternal()

	h := &numberHandlerTelnyx{
		reqHandler:      r,
		db:              db,
		requestExternal: reqExternal,
	}

	return h
}
