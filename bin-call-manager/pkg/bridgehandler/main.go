package bridgehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package bridgehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-call-manager/models/common"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/dbhandler"
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

		referenceType bridge.ReferenceType,
		referenceID uuid.UUID,
	) (*bridge.Bridge, error)
	Get(ctx context.Context, id string) (*bridge.Bridge, error)
	Delete(ctx context.Context, id string) (*bridge.Bridge, error)
	AddChannelID(ctx context.Context, id, channelID string) (*bridge.Bridge, error)
	RemoveChannelID(ctx context.Context, id, channelID string) (*bridge.Bridge, error)
	Destroy(ctx context.Context, id string) error

	ChannelKick(ctx context.Context, id string, channelID string) error
	ChannelJoin(ctx context.Context, id string, channelID string, role string, absorbDTMF bool, mute bool) error

	Start(ctx context.Context, asteriskID string, bridgeID string, bridgeName string, bridgeType []bridge.Type) (*bridge.Bridge, error)

	IsExist(ctx context.Context, id string) bool
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
	defaultExistTimeout = time.Second * 3
)

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
