package groupcallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package groupcallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-call-manager/models/common"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// GroupcallHandler is interface for service handle
type GroupcallHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*groupcall.Groupcall, error)
	UpdateAnswerCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) (*groupcall.Groupcall, error)
	Delete(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)

	Start(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		ownerType groupcall.OwnerType,
		ownerID uuid.UUID,
		flowID uuid.UUID,
		source *commonaddress.Address,
		destinations []commonaddress.Address,
		masterCallID uuid.UUID,
		masterGroupcallID uuid.UUID,
		ringMethod groupcall.RingMethod,
		answerMethod groupcall.AnswerMethod,
	) (*groupcall.Groupcall, error)
	AnswerCall(ctx context.Context, groupcallID uuid.UUID, answerCallID uuid.UUID) error
	AnswerGroupcall(ctx context.Context, id uuid.UUID, answerGroupcallID uuid.UUID) (*groupcall.Groupcall, error)
	Hangingup(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)

	HangupCall(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)
	HangupGroupcall(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error)

	IsGroupcallTypeAddress(destination *commonaddress.Address) bool

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
}

// groupcallHandler structure for service handle
type groupcallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
) GroupcallHandler {

	h := &groupcallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    requestHandler,
		notifyHandler: notifyHandler,
		db:            db,
	}

	return h
}
