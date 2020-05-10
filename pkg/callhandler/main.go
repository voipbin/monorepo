package callhandler

//go:generate mockgen -destination ./mock_callhandler_callhandler.go -package callhandler gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler CallHandler

import (
	"strings"
	"time"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// CallHandler is interface for service handle
type CallHandler interface {
	Start(cn *channel.Channel) error
	Hangup(cn *channel.Channel) error
	UpdateStatus(cn *channel.Channel) error
}

// callHandler structure for service handle
type callHandler struct {
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler
	confHandler conferencehandler.ConferenceHandler
}

// NewSvcHandler returns new service handler
func NewSvcHandler(r requesthandler.RequestHandler, d dbhandler.DBHandler) CallHandler {

	h := &callHandler{
		reqHandler:  r,
		db:          d,
		confHandler: conferencehandler.NewConferHandler(r, d),
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	date := time.Date(2018, 01, 12, 22, 51, 48, 324359102, time.UTC)

	res := date.String()
	res = strings.TrimSuffix(res, " +0000 UTC")

	return res
}
