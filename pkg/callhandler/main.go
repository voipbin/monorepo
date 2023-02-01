package callhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package callhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/confbridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/externalmediahandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/recordinghandler"
)

// CallHandler is interface for service handle
type CallHandler interface {
	ARIChannelDestroyed(ctx context.Context, cn *channel.Channel) error
	ARIChannelDtmfReceived(ctx context.Context, cn *channel.Channel, digit string, duration int) error
	ARIChannelLeftBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	ARIChannelStateChange(ctx context.Context, cn *channel.Channel) error
	ARIPlaybackFinished(ctx context.Context, cn *channel.Channel, playbackID string) error
	ARIStasisStart(ctx context.Context, cn *channel.Channel) error

	CallHealthCheck(ctx context.Context, id uuid.UUID, retryCount int, delay int)

	DigitsGet(ctx context.Context, id uuid.UUID) (string, error)
	DigitsSet(ctx context.Context, id uuid.UUID, digits string) error

	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*call.Call, error)
	Get(ctx context.Context, id uuid.UUID) (*call.Call, error)
	Delete(ctx context.Context, id uuid.UUID) (*call.Call, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status call.Status) (*call.Call, error)
	UpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*call.Call, error)
	UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error)
	UpdateConfbridgeID(ctx context.Context, id uuid.UUID, confbridgeID uuid.UUID) (*call.Call, error)

	CreateCallsOutgoing(ctx context.Context, customerID, flowID, masterCallID uuid.UUID, source commonaddress.Address, destinations []commonaddress.Address) ([]*call.Call, error)
	CreateCallOutgoing(ctx context.Context, id, customerID, flowID, activeflowID, masterCallID uuid.UUID, source commonaddress.Address, destination commonaddress.Address) (*call.Call, error)
	Start(ctx context.Context, cn *channel.Channel) error
	Hangup(ctx context.Context, cn *channel.Channel) error
	HangingUp(ctx context.Context, id uuid.UUID, reason call.HangupReason) (*call.Call, error)

	RecordingStart(
		ctx context.Context,
		id uuid.UUID,
		format recording.Format,
		endOfSilence int,
		endOfKey string,
		duration int,
	) (*call.Call, error)
	RecordingStop(ctx context.Context, id uuid.UUID) (*call.Call, error)

	ActionNext(ctx context.Context, c *call.Call) error
	ActionNextForce(ctx context.Context, c *call.Call) error
	ActionTimeout(ctx context.Context, callID uuid.UUID, a *fmaction.Action) error

	ChainedCallIDAdd(ctx context.Context, id, chainedCallID uuid.UUID) (*call.Call, error)
	ChainedCallIDRemove(ctx context.Context, id, chainedCallID uuid.UUID) (*call.Call, error)

	ExternalMediaStart(ctx context.Context, id uuid.UUID, externalHost string, encapsulation string, transport string, connectionType string, format string, direction string) (*call.Call, error)
	ExternalMediaStop(ctx context.Context, id uuid.UUID) (*call.Call, error)
}

// callHandler structure for service handle
type callHandler struct {
	utilHandler          utilhandler.UtilHandler
	reqHandler           requesthandler.RequestHandler
	db                   dbhandler.DBHandler
	notifyHandler        notifyhandler.NotifyHandler
	confbridgeHandler    confbridgehandler.ConfbridgeHandler
	channelHandler       channelhandler.ChannelHandler
	bridgeHandler        bridgehandler.BridgeHandler
	recordingHandler     recordinghandler.RecordingHandler
	externalMediaHandler externalmediahandler.ExternalMediaHandler
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
	defaultDialTimeout         = 60      // default outgoing dial timeout
	defaultTimeoutCallDuration = 3600000 // default call duration timeout. 1h
)

// list of variables
const (
	variableCallSourceName       = "voipbin.call.source.name"
	variableCallSourceDetail     = "voipbin.call.source.detail"
	variableCallSourceTarget     = "voipbin.call.source.target"
	variableCallSourceTargetName = "voipbin.call.source.target_name"
	variableCallSourceType       = "voipbin.call.source.type"

	variableCallDestinationName       = "voipbin.call.destination.name"
	variableCallDestinationDetail     = "voipbin.call.destination.detail"
	variableCallDestinationTarget     = "voipbin.call.destination.target"
	variableCallDestinationTargetName = "voipbin.call.destination.target_name"
	variableCallDestinationType       = "voipbin.call.destination.type"

	variableCallDirection    = "voipbin.call.direction"
	variableCallMasterCallID = "voipbin.call.master_call_id"
	variableCallDigits       = "voipbin.call.digits" // digit
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
func NewCallHandler(
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	confbridgeHandler confbridgehandler.ConfbridgeHandler,
	channelHandler channelhandler.ChannelHandler,
	bridgeHandler bridgehandler.BridgeHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalMediaHandler externalmediahandler.ExternalMediaHandler,
) CallHandler {

	h := &callHandler{
		utilHandler:          utilhandler.NewUtilHandler(),
		reqHandler:           requestHandler,
		notifyHandler:        notifyHandler,
		db:                   db,
		confbridgeHandler:    confbridgeHandler,
		channelHandler:       channelHandler,
		bridgeHandler:        bridgeHandler,
		recordingHandler:     recordingHandler,
		externalMediaHandler: externalMediaHandler,
	}

	return h
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
