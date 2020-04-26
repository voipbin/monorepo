package svchandler

//go:generate mockgen -destination ./mock_svchandler_svchandler.go -package svchandler gitlab.com/voipbin/bin-manager/call-manager/pkg/svchandler SVCHandler

import (
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arirequest"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
)

// SVCHandler is interface for service handle
type SVCHandler interface {
	Start(cn *channel.Channel) error
	Hangup(cn *channel.Channel) error
	UpdateStatus(cn *channel.Channel) error
}

// svcHandler structure for service handle
type svcHandler struct {
	reqHandler arirequest.RequestHandler
	db         dbhandler.DBHandler
}

// NewSvcHandler returns new service handler
func NewSvcHandler(r arirequest.RequestHandler, d dbhandler.DBHandler) SVCHandler {

	svchandler := &svcHandler{
		reqHandler: r,
		db:         d,
	}

	return svchandler
}
