package numberhandler

//go:generate mockgen -destination ./mock_numberhandler_numberhandler.go -package numberhandler -source ./main.go NumberHandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
)

// NumberHandler is interface for service handle
type NumberHandler interface {
	CreateOrderNumbers(userID uint64, numbers []string) ([]*models.Number, error)
	CreateOrderNumber(userID uint64, number string) (*models.Number, error)

	GetAvailableNumbers(countyCode string, limit uint) ([]*models.AvailableNumber, error)
	GetOrderNumber(ctx context.Context, id uuid.UUID) (*models.Number, error)
	GetOrderNumberByNumber(ctx context.Context, num string) (*models.Number, error)
	GetOrderNumbers(ctx context.Context, userID uint64, pageSize uint64, pageToken string) ([]*models.Number, error)

	ReleaseOrderNumbers(ctx context.Context, id uuid.UUID) (*models.Number, error)
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

// getCurTime return current utc time string
func getCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
