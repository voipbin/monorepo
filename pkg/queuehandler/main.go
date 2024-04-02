package queuehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package queuehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// List of default values
const (
	// defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
	defaultExecuteDelay = 1000 // 1000 ms(1 sec)
)

// QueueHandler interface
type QueueHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		name string,
		detail string,
		routingMethod queue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		waitTimeout int,
		serviceTimeout int,
	) (*queue.Queue, error)
	Delete(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	Execute(ctx context.Context, id uuid.UUID)
	Get(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queue.Queue, error)
	UpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		routingMethod queue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		waitTimeout int,
		serviceTimeout int,
	) (*queue.Queue, error)
	UpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*queue.Queue, error)
	UpdateRoutingMethod(ctx context.Context, id uuid.UUID, routingMEthod queue.RoutingMethod) (*queue.Queue, error)
	UpdateWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) (*queue.Queue, error)
	UpdateExecute(ctx context.Context, id uuid.UUID, execute queue.Execute) (*queue.Queue, error)

	AddWaitQueueCallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error)
	AddServiceQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error)
	RemoveServiceQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error)
	RemoveQueuecallID(ctx context.Context, id uuid.UUID, queuecallID uuid.UUID) (*queue.Queue, error)

	GetAgents(ctx context.Context, id uuid.UUID, status amagent.Status) ([]amagent.Agent, error)

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
}

type queueHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewQueueHandler return AgentHandler interface
func NewQueueHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) QueueHandler {
	return &queueHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,
	}
}
