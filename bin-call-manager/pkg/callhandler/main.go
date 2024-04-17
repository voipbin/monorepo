package callhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package callhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

// CallHandler is interface for service handle
type CallHandler interface {
	ARIChannelDestroyed(ctx context.Context, cn *channel.Channel) error
	ARIChannelDtmfReceived(ctx context.Context, cn *channel.Channel, digit string, duration int) error
	ARIChannelLeftBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	ARIChannelStateChange(ctx context.Context, cn *channel.Channel) error
	ARIPlaybackFinished(ctx context.Context, cn *channel.Channel, playbackID string) error
	ARIStasisStart(ctx context.Context, cn *channel.Channel) error

	HealthCheck(ctx context.Context, id uuid.UUID, retryCount int)

	DigitsGet(ctx context.Context, id uuid.UUID) (string, error)
	DigitsSet(ctx context.Context, id uuid.UUID, digits string) error

	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*call.Call, error)
	Get(ctx context.Context, id uuid.UUID) (*call.Call, error)
	Delete(ctx context.Context, id uuid.UUID) (*call.Call, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status call.Status) (*call.Call, error)
	UpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*call.Call, error)
	UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*call.Call, error)
	UpdateConfbridgeID(ctx context.Context, id uuid.UUID, confbridgeID uuid.UUID) (*call.Call, error)

	CreateCallsOutgoing(
		ctx context.Context,
		customerID uuid.UUID,
		flowID uuid.UUID,
		masterCallID uuid.UUID,
		source commonaddress.Address,
		destinations []commonaddress.Address,
		earlyExecution bool,
		connect bool,
	) ([]*call.Call, []*groupcall.Groupcall, error)
	CreateCallOutgoing(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		flowID uuid.UUID,
		activeflowID uuid.UUID,
		masterCallID uuid.UUID,
		groupcallID uuid.UUID,
		source commonaddress.Address,
		destination commonaddress.Address,
		earlyExecution bool,
		connect bool,
	) (*call.Call, error)
	Start(ctx context.Context, cn *channel.Channel) error
	Hangup(ctx context.Context, cn *channel.Channel) (*call.Call, error)
	HangingUp(ctx context.Context, id uuid.UUID, reason call.HangupReason) (*call.Call, error)

	Play(ctx context.Context, callID uuid.UUID, runNext bool, urls []string) error
	RecordingStart(
		ctx context.Context,
		id uuid.UUID,
		format recording.Format,
		endOfSilence int,
		endOfKey string,
		duration int,
	) (*call.Call, error)
	RecordingStop(ctx context.Context, id uuid.UUID) (*call.Call, error)
	Talk(ctx context.Context, callID uuid.UUID, runNext bool, text string, gender string, language string) error
	MediaStop(ctx context.Context, callID uuid.UUID) error
	HoldOn(ctx context.Context, id uuid.UUID) error
	HoldOff(ctx context.Context, id uuid.UUID) error
	MOHOn(ctx context.Context, id uuid.UUID) error
	MOHOff(ctx context.Context, id uuid.UUID) error
	MuteOn(ctx context.Context, id uuid.UUID, direction call.MuteDirection) error
	MuteOff(ctx context.Context, id uuid.UUID, direction call.MuteDirection) error
	SilenceOn(ctx context.Context, id uuid.UUID) error
	SilenceOff(ctx context.Context, id uuid.UUID) error

	ActionNext(ctx context.Context, c *call.Call) error
	ActionNextForce(ctx context.Context, c *call.Call) error
	ActionTimeout(ctx context.Context, callID uuid.UUID, a *fmaction.Action) error

	ChainedCallIDAdd(ctx context.Context, id, chainedCallID uuid.UUID) (*call.Call, error)
	ChainedCallIDRemove(ctx context.Context, id, chainedCallID uuid.UUID) (*call.Call, error)

	ExternalMediaStart(ctx context.Context, id uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*call.Call, error)
	ExternalMediaStop(ctx context.Context, id uuid.UUID) (*call.Call, error)

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
	EventFMActiveflowUpdated(ctx context.Context, a *fmactiveflow.Activeflow) error
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
	groupcallHandler     groupcallhandler.GroupcallHandler
}

// contextType
type contextType string

// List of contextType types.
const (
	contextTypeConference contextType = "conf"
	contextTypeCall       contextType = "call"
)

// List of default variables
const (
	defaultDialTimeout         = 60      // default outgoing dial timeout
	defaultTimeoutCallDuration = 3600000 // default call duration timeout. 1h

	defaultHealthMaxRetryCount = 2
	defaultHealthDelay         = 10000 // 10 seconds
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
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
	groupcallHandler groupcallhandler.GroupcallHandler,
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
		groupcallHandler:     groupcallHandler,
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
