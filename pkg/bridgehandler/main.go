package bridgehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package bridgehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// BridgeHandler is interface for service handle
type BridgeHandler interface {
	Create(
		ctx context.Context,

		asteriskID string,
		id string,
		name string,

		bridgeType bridge.Type,
		tech bridge.Tech,
		class string,
		creator string,

		videoMode string,
		videoSourceID string,

		channelIDs []string,

		referenceType bridge.ReferenceType,
		referenceID uuid.UUID,
	) (*bridge.Bridge, error)
	Get(ctx context.Context, id string) (*bridge.Bridge, error)
	Delete(ctx context.Context, id string) (*bridge.Bridge, error)
	AddChannelID(ctx context.Context, id, channelID string) (*bridge.Bridge, error)
	RemoveChannelID(ctx context.Context, id, channelID string) (*bridge.Bridge, error)
	GetWithTimeout(ctx context.Context, id string, timeout time.Duration) (*bridge.Bridge, error)
}

// bridgeHandler structure for service handle
type bridgeHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// list of default values
const (
	defaultDelayTimeout = time.Millisecond * 150
)

var (
	metricsNamespace = "call_manager"

	promBridgeCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bridge_create_total",
			Help:      "Total number of created bridge with reference_type.",
		},
		[]string{"reference_type"},
	)

	promBridgeDestroyedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "bridge_destroy_total",
			Help:      "Total number of destroyed bridge with reference_type.",
		},
		[]string{"reference_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promBridgeCreateTotal,
		promBridgeDestroyedTotal,
	)
}

// NewBridgeHandler returns new service handler
func NewBridgeHandler(r requesthandler.RequestHandler, n notifyhandler.NotifyHandler, db dbhandler.DBHandler) BridgeHandler {

	h := &bridgeHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    r,
		notifyHandler: n,
		db:            db,
	}

	return h
}
