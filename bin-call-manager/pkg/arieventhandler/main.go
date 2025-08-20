package arieventhandler

//go:generate mockgen -package arieventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	db "monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
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
	db          db.DBHandler
	cache       cachehandler.CacheHandler
	sockHandler sockhandler.SockHandler

	reqHandler           requesthandler.RequestHandler
	notifyHandler        notifyhandler.NotifyHandler
	callHandler          callhandler.CallHandler
	confbridgeHandler    confbridgehandler.ConfbridgeHandler
	channelHandler       channelhandler.ChannelHandler
	bridgeHandler        bridgehandler.BridgeHandler
	recordingHandler     recordinghandler.RecordingHandler
	externalmediaHandler externalmediahandler.ExternalMediaHandler
}

func init() {}

// NewEventHandler create EventHandler
func NewEventHandler(
	sock sockhandler.SockHandler,
	db db.DBHandler,
	cache cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	callHandler callhandler.CallHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	brideHandler bridgehandler.BridgeHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalmediaHandler externalmediahandler.ExternalMediaHandler,
) ARIEventHandler {
	h := &eventHandler{
		sockHandler:          sock,
		db:                   db,
		cache:                cache,
		reqHandler:           reqHandler,
		notifyHandler:        notifyHandler,
		callHandler:          callHandler,
		confbridgeHandler:    confbridgeHandler,
		channelHandler:       channelHandler,
		bridgeHandler:        brideHandler,
		recordingHandler:     recordingHandler,
		externalmediaHandler: externalmediaHandler,
	}

	return h
}
