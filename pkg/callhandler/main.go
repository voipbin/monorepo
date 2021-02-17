package callhandler

//go:generate mockgen -destination ./mock_callhandler_callhandler.go -package callhandler -source ./main.go CallHandler

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/action"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

// CallHandler is interface for service handle
type CallHandler interface {
	ARIChannelDestroyed(cn *channel.Channel) error
	ARIChannelDtmfReceived(cn *channel.Channel, digit string, duration int) error
	ARIChannelStateChange(cn *channel.Channel) error
	ARIPlaybackFinished(cn *channel.Channel, playbackID string) error
	ARIStasisStart(cn *channel.Channel, data map[string]interface{}) error

	CreateCallOutgoing(id uuid.UUID, userID uint64, flowID uuid.UUID, source call.Address, destination call.Address) (*call.Call, error)
	StartCallHandle(cn *channel.Channel, data map[string]interface{}) error
	Hangup(cn *channel.Channel) error
	HangupWithReason(ctx context.Context, c *call.Call, reason call.HangupReason, hangupBy call.HangupBy, timestamp string) error
	HangingUp(c *call.Call, cause ari.ChannelCause) error

	ActionNext(c *call.Call) error
	ActionTimeout(callID uuid.UUID, a *action.Action) error

	ChainedCallIDAdd(id, chainedCallID uuid.UUID) error
	ChainedCallIDRemove(id, chainedCallID uuid.UUID) error
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
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// getCurTime return current utc time string
func getCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
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
