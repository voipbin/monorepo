package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	context "context"
	"database/sql"
	"errors"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"

	queue "monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
)

// default variables
const (
	DefaultTimeStamp = "9999-01-01T00:00:00.000000Z" // DefaultTimeStamp default timestamp
)

// DBHandler interface
type DBHandler interface {
	// Queue operations
	QueueCreate(ctx context.Context, a *queue.Queue) error
	QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error)
	QueueList(ctx context.Context, size uint64, token string, filters map[queue.Field]any) ([]*queue.Queue, error)
	QueueUpdate(ctx context.Context, id uuid.UUID, fields map[queue.Field]any) error
	QueueDelete(ctx context.Context, id uuid.UUID) error

	// Queue specific operations
	QueueAddWaitQueueCallID(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueIncreaseTotalServicedCount(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueIncreaseTotalAbandonedCount(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueRemoveServiceQueueCall(ctx context.Context, id, queueCallID uuid.UUID) error
	QueueRemoveWaitQueueCall(ctx context.Context, id, queueCallID uuid.UUID) error

	// Queuecall operations
	QueuecallCreate(ctx context.Context, a *queuecall.Queuecall) error
	QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	QueuecallList(ctx context.Context, size uint64, token string, filters map[queuecall.Field]any) ([]*queuecall.Queuecall, error)
	QueuecallUpdate(ctx context.Context, id uuid.UUID, fields map[queuecall.Field]any) error
	QueuecallDelete(ctx context.Context, id uuid.UUID) error

	// Queuecall status operations
	QueuecallSetStatusConnecting(ctx context.Context, id uuid.UUID, serviceAgentID uuid.UUID) error
	QueuecallSetStatusService(ctx context.Context, id uuid.UUID, durationWaiting int, ts *time.Time) error
	QueuecallSetStatusAbandoned(ctx context.Context, id uuid.UUID, durationWaiting int, ts *time.Time) error
	QueuecallSetStatusDone(ctx context.Context, id uuid.UUID, durationService int, ts *time.Time) error
	QueuecallSetStatusWaiting(ctx context.Context, id uuid.UUID) error
	QueuecallSetStatusKicking(ctx context.Context, id uuid.UUID) error
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
