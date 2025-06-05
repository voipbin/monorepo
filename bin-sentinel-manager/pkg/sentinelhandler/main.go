package sentinelhandler

import (
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

type sentinelHandler struct {
	util          utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

type SentinelHandler interface {
	// AsteriskCrashHandle() error
}

func NewSentinelHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler utilhandler.UtilHandler,
) SentinelHandler {
	h := &sentinelHandler{
		util:          utilHandler,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
