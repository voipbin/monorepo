package callhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package callhandler -destination ./mock_callhandler_callhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// CallHandler is interface for service handle
type CallHandler interface {
	ARIChannelDestroyed(cn *channel.Channel) error
	ARIChannelDtmfReceived(cn *channel.Channel, digit string, duration int) error
	ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error
	ARIChannelStateChange(cn *channel.Channel) error
	ARIPlaybackFinished(cn *channel.Channel, playbackID string) error
	ARIStasisStart(cn *channel.Channel, data map[string]string) error

	CreateCallOutgoing(id uuid.UUID, userID uint64, flowID uuid.UUID, source address.Address, destination address.Address) (*call.Call, error)
	StartCallHandle(cn *channel.Channel, data map[string]string) error
	Hangup(cn *channel.Channel) error
	HangupWithReason(ctx context.Context, c *call.Call, reason call.HangupReason, hangupBy call.HangupBy, timestamp string) error
	HangingUp(c *call.Call, cause ari.ChannelCause) error

	ActionNext(c *call.Call) error
	ActionTimeout(callID uuid.UUID, a *action.Action) error

	ChainedCallIDAdd(id, chainedCallID uuid.UUID) error
	ChainedCallIDRemove(id, chainedCallID uuid.UUID) error

	ExternalMediaStart(callID uuid.UUID, isCallMedia bool, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*channel.Channel, error)
	ExternalMediaStop(callID uuid.UUID) error
}

// callHandler structure for service handle
type callHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
	confHandler   conferencehandler.ConferenceHandler
	notifyHandler notifyhandler.NotifyHandler
}

// contextType
type contextType string

// List of contextType types.
const (
	contextTypeConference contextType = "conf"
	contextTypeCall       contextType = "call"
)

// List of default values
const (
	defaultDialTimeout = 60 // default outgoing dial timeout
	defaultTimeStamp   = "9999-01-01 00:00:00.000000"
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

	promCallActionProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "call_action_process_time",
			Help:      "Process time of action execution",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type"},
	)

	promConferenceLeaveTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "conference_leave_total",
			Help:      "Total number of leaved calls from the conference with type.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promCallCreateTotal,
		promCallHangupTotal,
		promCallActionTotal,
		promCallActionProcessTime,
		promConferenceLeaveTotal,
	)
}

// NewCallHandler returns new service handler
func NewCallHandler(r requesthandler.RequestHandler, n notifyhandler.NotifyHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) CallHandler {

	h := &callHandler{
		reqHandler:    r,
		notifyHandler: n,
		db:            db,
		cache:         cache,
		confHandler:   conferencehandler.NewConferHandler(r, n, db, cache),
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
