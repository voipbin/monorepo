package externalmediahandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package externalmediahandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// ExternalMediaHandler defines
type ExternalMediaHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*externalmedia.ExternalMedia, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*externalmedia.ExternalMedia, error)
	Start(ctx context.Context, referenceType externalmedia.ReferenceType, referenceID uuid.UUID, insertMedia bool, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*externalmedia.ExternalMedia, error)
	Stop(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error)
}

// list of channel variables
const (
	ChannelValiableExternalMediaLocalPort    = "UNICASTRTP_LOCAL_PORT"
	ChannelValiableExternalMediaLocalAddress = "UNICASTRTP_LOCAL_ADDRESS"
)

const (
	defaultEncapsulation  = externalmedia.EncapsulationRTP
	defaultTransport      = externalmedia.TransportUDP
	defaultConnectionType = "client"
	defaultFormat         = "ulaw"
	defaultDirection      = "both" //
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
