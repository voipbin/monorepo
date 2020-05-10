package svchandler

//go:generate mockgen -destination ./mock_svchandler_svchandler.go -package svchandler gitlab.com/voipbin/bin-manager/call-manager/pkg/svchandler SVCHandler

import (
	"strings"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferhandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/db_handler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// SVCHandler is interface for service handle
type SVCHandler interface {
	Start(cn *channel.Channel) error
	Hangup(cn *channel.Channel) error
	UpdateStatus(cn *channel.Channel) error
}

// svcHandler structure for service handle
type svcHandler struct {
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler
	confHandler conferhandler.ConferenceHandler
}

// NewSvcHandler returns new service handler
func NewSvcHandler(r requesthandler.RequestHandler, d dbhandler.DBHandler) SVCHandler {

	svchandler := &svcHandler{
		reqHandler:  r,
		db:          d,
		confHandler: conferhandler.NewConferHandler(r, d),
	}

	return svchandler
}

// getCurTime return current utc time string
func getCurTime() string {
	date := time.Date(2018, 01, 12, 22, 51, 48, 324359102, time.UTC)

	res := date.String()
	res = strings.TrimSuffix(res, " +0000 UTC")

	return res
}
