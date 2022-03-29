package queuehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package queuehandler -destination ./mock_queuehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
)

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
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
	Get(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queue.Queue, error)
	Join(
		ctx context.Context,
		queueID uuid.UUID,
		referenceType queuecall.ReferenceType,
		referenceID uuid.UUID,
		referenceActiveflowID uuid.UUID,
		exitActionID uuid.UUID,
	) (*queuecall.Queuecall, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*queue.Queue, error)
	UpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*queue.Queue, error)
	UpdateRoutingMethod(ctx context.Context, id uuid.UUID, routingMEthod queue.RoutingMethod) (*queue.Queue, error)
	UpdateWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) (*queue.Queue, error)
}

type queueHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler

	queuecallHandler          queuecallhandler.QueuecallHandler
	queuecallReferenceHandler queuecallreferencehandler.QueuecallReferenceHandler
}

// NewQueueHandler return AgentHandler interface
func NewQueueHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	queuecallHandler queuecallhandler.QueuecallHandler,
	queuecallReferenceHandler queuecallreferencehandler.QueuecallReferenceHandler,
) QueueHandler {
	return &queueHandler{
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,

		queuecallHandler:          queuecallHandler,
		queuecallReferenceHandler: queuecallReferenceHandler,
	}
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
