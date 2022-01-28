package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	context "context"
	"database/sql"
	"errors"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	queue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

// default variables
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" // DefaultTimeStamp default timestamp
)

// DBHandler interface
type DBHandler interface {
	QueueAddQueueCallID(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueCreate(ctx context.Context, a *queue.Queue) error
	QueueDelete(ctx context.Context, id uuid.UUID) error
	QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	QueueGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queue.Queue, error)
	QueueIncreaseTotalServicedCount(ctx context.Context, id, queueCallID uuid.UUID, waittime time.Duration) error
	QueueIncreaseTotalAbandonedCount(ctx context.Context, id, queueCallID uuid.UUID, waittime time.Duration) error
	QueueRemoveServiceQueueCall(ctx context.Context, id, queueCallID uuid.UUID, serviceTime time.Duration) error
	QueueSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail, webhookURI, webhookMethod string) error
	QueueSetRoutingMethod(ctx context.Context, id uuid.UUID, routingMethod queue.RoutingMethod) error
	QueueSetTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error
	QueueSetWaitActionsAndTimeouts(ctx context.Context, id uuid.UUID, waitActions []fmaction.Action, waitTimeout, serviceTimeout int) error

	QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queuecall.Queuecall, error)
	QueuecallGetsByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*queuecall.Queuecall, error)
	QueuecallCreate(ctx context.Context, a *queuecall.Queuecall) error
	QueuecallDelete(ctx context.Context, id uuid.UUID, status queuecall.Status) error
	QueuecallSetServiceAgentID(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID) error
	QueuecallSetStatusService(ctx context.Context, id uuid.UUID) error

	QueuecallReferenceCreate(ctx context.Context, a *queuecallreference.QueuecallReference) error
	QueuecallReferenceDelete(ctx context.Context, id uuid.UUID) error
	QueuecallReferenceGet(ctx context.Context, id uuid.UUID) (*queuecallreference.QueuecallReference, error)
	QueuecallReferenceSetCurrentQueuecallID(ctx context.Context, id, queuecallID uuid.UUID) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}

// GetCurTime return current utc time string
func GetCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
