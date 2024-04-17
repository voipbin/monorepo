package outdialtargetcallhandler

import (
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

type outdialTargetCallHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// OutdialTargetCallHandler interface
type OutdialTargetCallHandler interface {
}

// NewOutdialTargetCallHandler returns OutdialTargetCallHandler
func NewOutdialTargetCallHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) OutdialTargetCallHandler {
	h := &outdialTargetCallHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
