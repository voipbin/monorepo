package channelhandler

//go:generate mockgen -package channelhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-call-manager/models/common"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
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

		sourceName string,
		sourceNumber string,
		destinationName string,
		destinationNumber string,

		state ari.ChannelState,
	) (*channel.Channel, error)
	Get(ctx context.Context, id string) (*channel.Channel, error)
	GetChannelsForRecovery(
		ctx context.Context,
		asteriskID string,
		channelType channel.Type,
		startTime string,
		endTime string,
		size uint64,
	) ([]*channel.Channel, error)
	Delete(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error)
	SetDataItem(ctx context.Context, id string, key string, value interface{}) error
	SetSIPTransport(ctx context.Context, id string, transport channel.SIPTransport) error
	SetDirection(ctx context.Context, id string, direction channel.Direction) error

	ARIChannelStateChange(ctx context.Context, e *ari.ChannelStateChange) (*channel.Channel, error)
	ARIStasisStart(ctx context.Context, e *ari.StasisStart) (*channel.Channel, error)

	AddressGetSource(cn *channel.Channel, addressType commonaddress.Type) *commonaddress.Address
	AddressGetDestination(cn *channel.Channel, addressType commonaddress.Type) *commonaddress.Address
	AddressGetDestinationWithoutSpecificType(cn *channel.Channel) *commonaddress.Address

	UpdateStasisName(ctx context.Context, id string, stasisName string) (*channel.Channel, error)
	UpdateState(ctx context.Context, id string, state ari.ChannelState) (*channel.Channel, error)
	UpdateBridgeID(ctx context.Context, id string, bridgeID string) (*channel.Channel, error)
	UpdatePlaybackID(ctx context.Context, id string, playbackID string) (*channel.Channel, error)

	Answer(ctx context.Context, id string) error

	StartChannelWithBaseChannel(ctx context.Context, baseChannelID string, id string, appArgs string, endpoint string, otherChannelID string, originator string, formats string, variables map[string]string) (*channel.Channel, error)
	StartChannel(ctx context.Context, asteriskID string, id string, appArgs string, endpoint string, otherChannelID string, originator string, formats string, variables map[string]string) (*channel.Channel, error)
	StartSnoop(ctx context.Context, id string, snoopID string, appArgs string, spy channel.SnoopDirection, whisper channel.SnoopDirection) (*channel.Channel, error)
	StartExternalMedia(ctx context.Context, asteriskID string, id string, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string, data string, variables map[string]string) (*channel.Channel, error)

	DTMFSend(ctx context.Context, id string, digit string, duration int, before int, between int, after int) error

	Hangup(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error)
	HangingUp(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error)
	HangingUpWithAsteriskID(ctx context.Context, asteriskID string, id string, cause ari.ChannelCause) error
	HangingUpWithDelay(ctx context.Context, id string, cause ari.ChannelCause, delay int) (*channel.Channel, error)

	HealthCheck(ctx context.Context, channelID string, retryCount int)

	HoldOn(ctx context.Context, id string) error
	HoldOff(ctx context.Context, id string) error

	MOHOn(ctx context.Context, id string) error
	MOHOff(ctx context.Context, id string) error

	MuteOn(ctx context.Context, id string, direction channel.MuteDirection) error
	MuteOff(ctx context.Context, id string, direction channel.MuteDirection) error

	SilenceOn(ctx context.Context, id string) error
	SilenceOff(ctx context.Context, id string) error

	Continue(ctx context.Context, id string, context string, exten string, priority int, label string) error

	Play(ctx context.Context, id string, actionID uuid.UUID, medias []string, language string) error
	PlaybackStop(ctx context.Context, id string) error

	Redirect(ctx context.Context, id string, context string, exten string, priority string) error
	Record(ctx context.Context, id string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error
	Ring(ctx context.Context, id string) error

	VariableSet(ctx context.Context, id string, key string, value string) error

	Dial(ctx context.Context, id string, caller string, timeout int) error
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
	defaultDelayTimeout        = time.Millisecond * 150
	defaultExistTimeout        = time.Second * 3
	defaultHealthMaxRetryCount = 2
	defaultHealthDelay         = 10000 // 10 seconds
)

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
