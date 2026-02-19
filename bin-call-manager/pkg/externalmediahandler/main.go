package externalmediahandler

//go:generate mockgen -package externalmediahandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// ExternalMediaHandler defines
type ExternalMediaHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*externalmedia.ExternalMedia, error)
	List(ctx context.Context, size uint64, token string, filters map[externalmedia.Field]any) ([]*externalmedia.ExternalMedia, error)
	Start(
		ctx context.Context,
		id uuid.UUID,
		typ externalmedia.Type,
		referenceType externalmedia.ReferenceType,
		referenceID uuid.UUID,
		externalHost string,
		encapsulation externalmedia.Encapsulation,
		transport externalmedia.Transport,
		connectionType string,
		format string,
		directionListen externalmedia.Direction,
		directionSpeak externalmedia.Direction,
	) (*externalmedia.ExternalMedia, error)
	Stop(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error)

	ARIPlaybackFinished(ctx context.Context, cn *bridge.Bridge, e *ari.PlaybackFinished) error
}

// list of channel variables
const (
	ChannelValiableExternalMediaLocalPort    = "UNICASTRTP_LOCAL_PORT"
	ChannelValiableExternalMediaLocalAddress = "UNICASTRTP_LOCAL_ADDRESS"
)

const (
	defaultEncapsulation        = externalmedia.EncapsulationRTP
	defaultTransport            = externalmedia.TransportUDP
	defaultConnectionType       = "client"
	defaultFormat               = "ulaw"
	defaultDirection            = "both" //
	defaultSilencePlaybackMedia = "sound:silence_slin16_8000_1m"
)

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

	// external_media_start_total counts external media starts by reference type and encapsulation.
	promExternalMediaStartTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "external_media_start_total",
			Help:      "Total number of external media starts by reference type and encapsulation.",
		},
		[]string{"reference_type", "encapsulation"},
	)

	// external_media_stop_total counts external media stops by reference type.
	promExternalMediaStopTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "external_media_stop_total",
			Help:      "Total number of external media stops by reference type.",
		},
		[]string{"reference_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promExternalMediaStartTotal,
		promExternalMediaStopTotal,
	)
}

type externalMediaHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	channelHandler channelhandler.ChannelHandler
	bridgeHandler  bridgehandler.BridgeHandler
}

// NewExternalMediaHandler returns new service handler
func NewExternalMediaHandler(
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	channelHandler channelhandler.ChannelHandler,
	bridgeHandler bridgehandler.BridgeHandler,
) ExternalMediaHandler {

	h := &externalMediaHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		reqHandler:     requestHandler,
		notifyHandler:  notifyHandler,
		db:             db,
		channelHandler: channelHandler,
		bridgeHandler:  bridgeHandler,
	}

	return h
}
