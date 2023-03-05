package groupcallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package groupcallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// GroupcallHandler is interface for service handle
type GroupcallHandler interface {
	Start(
		ctx context.Context,
		customerID uuid.UUID,
		source *commonaddress.Address,
		destinations []commonaddress.Address,
		flowID uuid.UUID,
		masterCallID uuid.UUID,
		ringMethod groupcall.RingMethod,
		answerMethod groupcall.AnswerMethod,
		connect bool,
	) (*groupcall.Groupcall, error)
}

// groupcallHandler structure for service handle
type groupcallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
	callHandler   callhandler.CallHandler
}

var (
	metricsNamespace = "call_manager"

	promGroupcallCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "groupcall_create_total",
			Help:      "Total number of created call direction with type.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promGroupcallCreateTotal,
	)
}

// NewGroupcallHandler returns new service handler
func NewGroupcallHandler(
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	callHandler callhandler.CallHandler,
) GroupcallHandler {

	h := &groupcallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    requestHandler,
		notifyHandler: notifyHandler,
		db:            db,
		callHandler:   callHandler,
	}

	return h
}
