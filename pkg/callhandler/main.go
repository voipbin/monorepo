package callhandler

//go:generate mockgen -destination ./mock_callhandler_callhandler.go -package callhandler -source ./main.go CallHandler

import (
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/action"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// CallHandler is interface for service handle
type CallHandler interface {
	ARIChannelDestroyed(cn *channel.Channel) error
	ARIChannelDtmfReceived(cn *channel.Channel, digit string, duration int) error
	ARIStasisStart(cn *channel.Channel) error

	Start(cn *channel.Channel) error
	Hangup(cn *channel.Channel) error
	UpdateStatus(cn *channel.Channel) error

	ActionNext(c *call.Call) error
	ActionTimeout(callID uuid.UUID, a *action.Action) error
}

// callHandler structure for service handle
type callHandler struct {
	reqHandler  requesthandler.RequestHandler
	db          dbhandler.DBHandler
	cache       cachehandler.CacheHandler
	confHandler conferencehandler.ConferenceHandler
}

// contextType
type contextType string

// List of contextType types.
const (
	contextTypeConference contextType = "conf"
	contextTypeCall       contextType = "call"
)

var (
	metricsNamespace = "call_manager"

	promCallCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "call_create_total",
			Help:      "Total number of created call direction with type.",
		},
		[]string{"direction", "type"},
	)

	promCallHangupTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "call_hangup_total",
			Help:      "Total number of hungup call direction with type and reason.",
		},
		[]string{"direction", "type", "reason"},
	)

	promCallActionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "call_action_total",
			Help:      "Total number of executed actions.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promCallCreateTotal,
		promCallHangupTotal,
		promCallActionTotal,
	)
}

// NewCallHandler returns new service handler
func NewCallHandler(r requesthandler.RequestHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) CallHandler {

	h := &callHandler{
		reqHandler:  r,
		db:          db,
		cache:       cache,
		confHandler: conferencehandler.NewConferHandler(r, db, cache),
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

// getContextType returns CONTEXT's type
func getContextType(message interface{}) contextType {
	if message == nil {
		return contextTypeCall
	}

	tmp := strings.Split(message.(string), "-")[0]
	switch tmp {
	case string(contextTypeConference):
		return contextTypeConference
	default:
		return contextTypeCall
	}
}
