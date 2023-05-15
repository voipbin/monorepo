package arieventhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package arieventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	db "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/recordinghandler"
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
	channelHandler    channelhandler.ChannelHandler
	bridgeHandler     bridgehandler.BridgeHandler
	recordingHandler  recordinghandler.RecordingHandler
}

func init() {}

// NewEventHandler create EventHandler
func NewEventHandler(
	sock rabbitmqhandler.Rabbit,
	db db.DBHandler,
	cache cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	brideHandler bridgehandler.BridgeHandler,
	recordingHandler recordinghandler.RecordingHandler,
) ARIEventHandler {
	h := &eventHandler{
		rabbitSock:        sock,
		db:                db,
		cache:             cache,
		reqHandler:        reqHandler,
		notifyHandler:     notifyHandler,
		callHandler:       callHandler,
		confbridgeHandler: confbridgeHandler,
		channelHandler:    channelHandler,
		bridgeHandler:     brideHandler,
		recordingHandler:  recordingHandler,
	}

	return h
}
