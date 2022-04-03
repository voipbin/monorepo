package outdialtargetcallhandler

import (
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/dbhandler"
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
