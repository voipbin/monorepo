package numberhandlertelnyx

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandlertelnyx -destination ./mock_numberhandler_numberhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

// NumberHandler is interface for service handle
type NumberHandler interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)
	CreateOrderNumbers(userID uint64, numbs []string) ([]*number.Number, error)
	ReleaseOrderNumber(ctx context.Context, numb *number.Number) (*number.Number, error)
}

// numberHandler structure for service handle
type numberHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler
}

// telnyx const variables
const (
	ConnectionID string = "1526401767787464160" // telnyx's voipbin connection id
)

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000"
)

var (
	metricsNamespace = "number_manager"

	promNumberCreateTotal = prometheus.NewCounterVec(
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
		promNumberCreateTotal,
	)
}

// NewNumberHandler returns new service handler
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) NumberHandler {

	h := &numberHandler{
		reqHandler: r,
		db:         db,
		cache:      cache,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// getCurTime return current utc time string
func getCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
