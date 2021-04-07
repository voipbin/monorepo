package stthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package stthandler -destination ./mock_stthandler_stthandler.go -source main.go -build_flags=-mod=mod

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/requesthandler"
)

// STTHandler is interface for service handle
type STTHandler interface {
	// GetAvailableNumbers(countyCode string, limit uint) ([]*availablenumber.AvailableNumber, error)

	// CreateNumbers(userID uint64, numbs []string) ([]*number.Number, error)
	// CreateNumber(userID uint64, numb string) (*number.Number, error)
	// GetNumber(ctx context.Context, id uuid.UUID) (*number.Number, error)
	// GetNumberByNumber(ctx context.Context, num string) (*number.Number, error)
	// GetNumbers(ctx context.Context, userID uint64, pageSize uint64, pageToken string) ([]*number.Number, error)

	// ReleaseNumber(ctx context.Context, id uuid.UUID) (*number.Number, error)

	// RemoveNumbersFlowID(ctx context.Context, flowID uuid.UUID) error

	// UpdateNumber(ctx context.Context, numb *number.Number) (*number.Number, error)
}

// sttHandler structure for service handle
type sttHandler struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler
	cache      cachehandler.CacheHandler
}

var (
	metricsNamespace = "stt_manager"

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

// NewSTTHandler returns new service handler
func NewSTTHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) STTHandler {

	h := &sttHandler{
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
