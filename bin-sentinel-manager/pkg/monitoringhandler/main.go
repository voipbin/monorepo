package monitoringhandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

type monitoringHandler struct {
	util          utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

type MonitoringHandler interface {
	Run(ctx context.Context, selectors map[string][]string) error
}

func NewMonitoringHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler utilhandler.UtilHandler,
) MonitoringHandler {
	h := &monitoringHandler{
		util:          utilHandler,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
