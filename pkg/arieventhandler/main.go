package arieventhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package arieventhandler -destination ./mock_arieventhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	db "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

// ARIEventHandler intreface for ARI request handler
type ARIEventHandler interface {
	EventHandlerContactStatusChange(ctx context.Context, evt interface{}) error

	EventHandlerBridgeCreated(ctx context.Context, evt interface{}) error
	EventHandlerBridgeDestroyed(ctx context.Context, evt interface{}) error

	EventHandlerChannelCreated(ctx context.Context, evt interface{}) error
	EventHandlerChannelDestroyed(ctx context.Context, evt interface{}) error
	EventHandlerChannelVarset(ctx context.Context, evt interface{}) error
	EventHandlerChannelStateChange(ctx context.Context, evt interface{}) error
	EventHandlerChannelEnteredBridge(ctx context.Context, evt interface{}) error
	EventHandlerChannelLeftBridge(ctx context.Context, evt interface{}) error
	EventHandlerChannelDtmfReceived(ctx context.Context, evt interface{}) error

	EventHandlerStasisStart(ctx context.Context, evt interface{}) error
	EventHandlerStasisEnd(ctx context.Context, evt interface{}) error

	EventHandlerRecordingStarted(ctx context.Context, evt interface{}) error
	EventHandlerRecordingFinished(ctx context.Context, evt interface{}) error

	EventHandlerPlaybackStarted(ctx context.Context, evt interface{}) error
	EventHandlerPlaybackFinished(ctx context.Context, evt interface{}) error
}

type eventHandler struct {
	db         db.DBHandler
	cache      cachehandler.CacheHandler
	rabbitSock rabbitmqhandler.Rabbit

	reqHandler        requesthandler.RequestHandler
	notifyHandler     notifyhandler.NotifyHandler
	callHandler       callhandler.CallHandler
	confbridgeHandler confbridgehandler.ConfbridgeHandler
}

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

var (
	metricsNamespace = "call_manager"

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
		promChannelTransportAndDirection,
	)

}

// NewEventHandler create EventHandler
func NewEventHandler(
	sock rabbitmqhandler.Rabbit,
	db db.DBHandler,
	cache cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
) ARIEventHandler {
	h := &eventHandler{
		rabbitSock:        sock,
		db:                db,
		cache:             cache,
		reqHandler:        reqHandler,
		notifyHandler:     notifyHandler,
		callHandler:       callHandler,
		confbridgeHandler: confbridgeHandler,
	}

	return h
}

// contextType
type contextType string

// List of contextType types.
const (
	contextTypeConference contextType = "conf"
	contextTypeCall       contextType = "call"
)

const defaultExistTimeout = time.Second * 3

// getContextType returns CONTEXT's type
func getContextType(message interface{}) contextType {
	if message == nil {
		return contextTypeCall
	}

	tmp := strings.Split(message.(string), "-")[0]
	switch tmp {
	case string(contextTypeConference):
		return contextTypeConference
	default:
		return contextTypeCall
	}
}
