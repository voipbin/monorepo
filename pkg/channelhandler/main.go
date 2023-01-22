package channelhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package channelhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// ChannelHandler is interface for service handle
type ChannelHandler interface {
	Create(
		ctx context.Context,

		id string,
		asteriskID string,
		name string,
		channelType channel.Type,
		tech channel.Tech,

		sipCallID string,
		sipTransport channel.SIPTransport,

		sourceName string,
		sourceNumber string,
		destinationName string,
		destinationNumber string,

		state ari.ChannelState,
		data map[string]interface{},
		stasisName string,
		stasisData map[string]string,
		bridgeID string,
		playbackID string,

		dialResult string,
		hangupCause ari.ChannelCause,

		direction channel.Direction,
	) (*channel.Channel, error)
	Get(ctx context.Context, id string) (*channel.Channel, error)
	GetWithTimeout(ctx context.Context, id string, timeout time.Duration) (*channel.Channel, error)
	Delete(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error)
	SetDataItem(ctx context.Context, id string, key string, value interface{}) error
	SetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error
	SetDirection(ctx context.Context, id string, direction channel.Direction) error
	SetSIPCallID(ctx context.Context, id string, sipCallID string) error
	SetType(ctx context.Context, id string, channelType channel.Type) error

	UpdateState(ctx context.Context, id string, state ari.ChannelState) (*channel.Channel, error)
	UpdateBridgeID(ctx context.Context, id string, bridgeID string) (*channel.Channel, error)

	Hangup(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error)
	HangingUp(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error)
	HangingUpWithAsteriskID(ctx context.Context, asteriskID string, id string, cause ari.ChannelCause) error

	HealthCheck(ctx context.Context, channelID string, retryCount int, retryCountMax int, delay int)
}

// channelHandler structure for service handle
type channelHandler struct {
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

	promChannelCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "channel_create_total",
			Help:      "Total number of created channel direction with tech.",
		},
		[]string{"direction", "tech"},
	)

	promChannelDestroyedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "channel_hangup_total",
			Help:      "Total number of destroyed channel direction with tech and reason.",
		},
		[]string{"direction", "type", "reason"},
	)

	promChannelTransportAndDirection = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "channel_transport_direction_total",
			Help:      "Total number of channel's transport and direction.",
		},
		[]string{"transport", "direction"},
	)
)

func init() {
	prometheus.MustRegister(
		promChannelCreateTotal,
		promChannelDestroyedTotal,
		promChannelTransportAndDirection,
	)
}

// NewChannelHandler returns new service handler
func NewChannelHandler(r requesthandler.RequestHandler, n notifyhandler.NotifyHandler, db dbhandler.DBHandler) ChannelHandler {

	h := &channelHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    r,
		notifyHandler: n,
		db:            db,
	}

	return h
}
