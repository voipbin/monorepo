package activeflowhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package activeflowhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cmcustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/stackhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/variablehandler"
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
	stackHandler    stackhandler.StackHandler
}

// ActiveflowHandler defines
type ActiveflowHandler interface {
	Create(ctx context.Context, id uuid.UUID, referenceType activeflow.ReferenceType, referenceID uuid.UUID, flowID uuid.UUID) (*activeflow.Activeflow, error)
	Delete(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	Get(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)
	Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*activeflow.Activeflow, error)

	PushActions(ctx context.Context, id uuid.UUID, actions []action.Action) (*activeflow.Activeflow, error)

	SetForwardActionID(ctx context.Context, callID uuid.UUID, actionID uuid.UUID, forwardNow bool) error
	Stop(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error)

	Execute(ctx context.Context, id uuid.UUID) error
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

	stackHandler := stackhandler.NewStackHandler()

	return &activeflowHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		actionHandler:   actionHandler,
		variableHandler: variableHandler,
		stackHandler:    stackHandler,
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
)

func init() {
	prometheus.MustRegister(
		promActionExecuteDuration,
	)
}
