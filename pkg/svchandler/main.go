package svchandler

import (
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arirequest"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
)

// SVCHandler is interface for service handle
type SVCHandler interface {
	StasisStart(e *ari.StasisStart) error
}

// svcHandler structure for service handle
type svcHandler struct {
	reqHandler arirequest.RequestHandler
	db         dbhandler.DBHandler
}

// NewServiceHandler returns new service handler
func NewServiceHandler(r arirequest.RequestHandler, d dbhandler.DBHandler) SVCHandler{

	svchandler := &svcHandler {
		reqHandler: r,
		db: d,
	}

	return svchandler
}