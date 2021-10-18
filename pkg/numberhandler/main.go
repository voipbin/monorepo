package numberhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package numberhandler -destination ./mock_numberhandler_numberhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

// NumberHandler is interface for service handle
type NumberHandler interface {
	GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)

	CreateNumbers(userID uint64, numbs []string) ([]*number.Number, error)
	CreateNumber(userID uint64, numb string) (*number.Number, error)
	GetNumber(ctx context.Context, id uuid.UUID) (*number.Number, error)
	GetNumberByNumber(ctx context.Context, num string) (*number.Number, error)
	GetNumbers(ctx context.Context, userID uint64, pageSize uint64, pageToken string) ([]*number.Number, error)

	ReleaseNumber(ctx context.Context, id uuid.UUID) (*number.Number, error)

	RemoveNumbersFlowID(ctx context.Context, flowID uuid.UUID) error

	UpdateNumber(ctx context.Context, numb *number.Number) (*number.Number, error)
}

// numberHandler structure for service handle
type numberHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler

	numHandlerTelnyx numberhandlertelnyx.NumberHandler
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
func NewNumberHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) NumberHandler {

	nHandlerTelnyx := numberhandlertelnyx.NewNumberHandler(r, db, cache)

	h := &numberHandler{
		reqHandler: r,
		db:         db,
		cache:      cache,

		numHandlerTelnyx: nHandlerTelnyx,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
