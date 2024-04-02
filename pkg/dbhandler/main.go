package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	context "context"
	"database/sql"
	"errors"

	uuid "github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	queue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

// default variables
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" // DefaultTimeStamp default timestamp
)

// DBHandler interface
type DBHandler interface {
	QueueAddWaitQueueCallID(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueCreate(ctx context.Context, a *queue.Queue) error
	QueueDelete(ctx context.Context, id uuid.UUID) error
	QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	QueueGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queue.Queue, error)
	QueueIncreaseTotalServicedCount(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueIncreaseTotalAbandonedCount(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueRemoveServiceQueueCall(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueRemoveWaitQueueCall(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueSetBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		routingMethod queue.RoutingMethod,
		tagIDs []uuid.UUID,
		waitActions []fmaction.Action,
		waitTimeout int,
		serviceTimeout int,
	) error
	QueueSetRoutingMethod(ctx context.Context, id uuid.UUID, routingMethod queue.RoutingMethod) error
	QueueSetTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error
	QueueSetExecute(ctx context.Context, id uuid.UUID, execute queue.Execute) error
	QueueSetWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) error

	QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queuecall.Queuecall, error)
	// QueuecallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*queuecall.Queuecall, error)
	QueuecallCreate(ctx context.Context, a *queuecall.Queuecall) error
	QueuecallDelete(ctx context.Context, id uuid.UUID) error
	QueuecallSetStatusConnecting(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID) error
	QueuecallSetStatusService(ctx context.Context, id uuid.UUID, durationWaiting int, ts string) error
	QueuecallSetStatusAbandoned(ctx context.Context, id uuid.UUID, durationWaiting int, ts string) error
	QueuecallSetStatusDone(ctx context.Context, id uuid.UUID, durationService int, ts string) error
	QueuecallSetStatusWaiting(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}
