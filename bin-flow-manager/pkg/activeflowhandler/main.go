package activeflowhandler

//go:generate mockgen -package activeflowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

const (
	maxActiveFlowExecuteCount = 100
)

// activeflowHandler defines
type activeflowHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	actionHandler   actionhandler.ActionHandler
	variableHandler variablehandler.VariableHandler
	stackmapHandler stackmaphandler.StackmapHandler
}

// list of variables
const (
	variableActiveflowID                    = "voipbin.activeflow.id"
	variableActiveflowReferenceType         = "voipbin.activeflow.reference_type"
	variableActiveflowReferenceID           = "voipbin.activeflow.reference_id"
	variableActiveflowReferenceActiveflowID = "voipbin.activeflow.reference_activeflow_id"
	variableActiveflowFlowID                = "voipbin.activeflow.flow_id"
	variableActiveflowCompleteCount         = "voipbin.activeflow.complete_count" // indicates how many times the activeflow has completed(by on complete flow id)
)

const (
	maxNextActionLoopCount     = 1000 // max loop count for next action
	maxActiveflowCompleteCount = 5    // max loop count for activeflow (by on complete flow id)
)

// ActiveflowHandler defines
type ActiveflowHandler interface {
	Create(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		referenceType activeflow.ReferenceType,
		referenceID uuid.UUID,
		referenceActiveflowID uuid.UUID,
		flowID uuid.UUID,
	) (*activeflow.Activeflow, error)
	Delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	Get(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	List(ctx context.Context, token string, size uint64, filters map[activeflow.Field]any) ([]*activeflow.Activeflow, error)

	PopStackWithStackID(ctx context.Context, af *activeflow.Activeflow, stackID uuid.UUID) error
	PushActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*activeflow.Activeflow, error)
	AddActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*activeflow.Activeflow, error)

	SetForwardActionID(ctx context.Context, callID uuid.UUID, actionID uuid.UUID, forwardNow bool) error
	Stop(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	ServiceStop(ctx context.Context, id uuid.UUID, serviceID uuid.UUID) error

	Execute(ctx context.Context, id uuid.UUID) error
	ExecuteContinue(ctx context.Context, activeflowID uuid.UUID, caID uuid.UUID) error
	ExecuteNextAction(ctx context.Context, callID uuid.UUID, caID uuid.UUID) (*action.Action, error)

	EventCallHangup(ctx context.Context, c *cmcall.Call) error
	EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error
}

// NewActiveflowHandler returns new ActiveflowHandler
func NewActiveflowHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	actionHandler actionhandler.ActionHandler,
	variableHandler variablehandler.VariableHandler,
) ActiveflowHandler {

	stackHandler := stackmaphandler.NewStackmapHandler()

	return &activeflowHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		actionHandler:   actionHandler,
		variableHandler: variableHandler,
		stackmapHandler: stackHandler,
	}
}

var (
	metricsNamespace = "flow_manager"

	promActionExecuteDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "action_exeucte_duration",
			Help:      "Execute duration of action",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type"},
	)

	promActiveflowCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_created_total",
			Help:      "Total number of activeflows created",
		},
		[]string{"reference_type"},
	)

	promActiveflowEndedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_ended_total",
			Help:      "Total number of activeflows ended",
		},
		[]string{"reference_type"},
	)

	// promActiveflowRunning tracks in-process activeflows only.
	// It resets to 0 on service restart and does not reflect database state.
	promActiveflowRunning = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_running",
			Help:      "Number of currently running activeflows",
		},
		[]string{"reference_type"},
	)

	promActionExecutedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "action_executed_total",
			Help:      "Total number of actions executed",
		},
		[]string{"type"},
	)

	// promActionErrorTotal tracks only fatal action errors that stop the flow.
	// Non-critical actions (email_send, webhook_send, conversation_send, etc.)
	// swallow errors and continue the flow, so they are not counted here.
	promActionErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "action_error_total",
			Help:      "Total number of action execution errors that stopped the flow",
		},
		[]string{"type"},
	)

	promActiveflowDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_duration_seconds",
			Help:      "Total duration of activeflow from creation to end",
			Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"reference_type"},
	)

	promActionDispatchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "action_dispatch_total",
			Help:      "Total number of actions dispatched to external services",
		},
		[]string{"target", "type"},
	)

	promActiveflowExecuteIterations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "activeflow_execute_iterations",
			Help:      "Number of action iterations per execute loop",
			Buckets:   []float64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000},
		},
		[]string{"reference_type"},
	)
)

// actionDispatchTarget maps action types to the service they dispatch to.
var actionDispatchTarget = map[action.Type]string{
	action.TypeAISummary:          "ai-manager",
	action.TypeAITalk:             "ai-manager",
	action.TypeAITask:             "ai-manager",
	action.TypeConferenceJoin:     "conference-manager",
	action.TypeConnect:            "call-manager",
	action.TypeConversationSend:   "conversation-manager",
	action.TypeEmailSend:          "email-manager",
	action.TypeMessageSend:        "message-manager",
	action.TypeQueueJoin:          "queue-manager",
	action.TypeTranscribeRecording: "transcribe-manager",
	action.TypeTranscribeStart:    "transcribe-manager",
	action.TypeWebhookSend:        "webhook-manager",
}

func init() {
	prometheus.MustRegister(
		promActionExecuteDuration,
		promActiveflowCreatedTotal,
		promActiveflowEndedTotal,
		promActiveflowRunning,
		promActionExecutedTotal,
		promActionErrorTotal,
		promActiveflowDurationSeconds,
		promActionDispatchTotal,
		promActiveflowExecuteIterations,
	)
}
