package externalmediahandler

//go:generate mockgen -package externalmediahandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

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
