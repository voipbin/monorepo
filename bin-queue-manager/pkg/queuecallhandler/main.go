package queuecallhandler

//go:generate mockgen -package queuecallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

var (
	metricsNamespace = "queue_manager"

	promQueuecallCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "queuecall_create_total",
			Help:      "Total number of queuecalls created.",
		},
	)

	promQueuecallDoneTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "queuecall_done_total",
			Help:      "Total number of queuecalls completed successfully.",
		},
	)

	promQueuecallAbandonedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "queuecall_abandoned_total",
			Help:      "Total number of queuecalls abandoned.",
		},
	)

	promQueuecallWaitingDurationSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "queuecall_waiting_duration_seconds",
			Help:      "Duration of queuecall waiting time in seconds before service.",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
	)
)

func init() {
	prometheus.MustRegister(
		promQueuecallCreateTotal,
		promQueuecallDoneTotal,
		promQueuecallAbandonedTotal,
		promQueuecallWaitingDurationSeconds,
	)
}

const (
	defaultHealthCheckMaxRetryCount = 2
	defaultHealthCheckDelay         = 5000 // 5 seconds
)

// QueuecallHandler interface
type QueuecallHandler interface {
	Create(
		ctx context.Context,
		q *queue.Queue,
		id uuid.UUID,
		referenceType queuecall.ReferenceType,
		referenceID uuid.UUID,
		referenceActiveflowID uuid.UUID,
		forwardActionID uuid.UUID,
		conferenceID uuid.UUID,
		source commonaddress.Address,
	) (*queuecall.Queuecall, error)
	Get(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	List(ctx context.Context, size uint64, token string, filters map[queuecall.Field]any) ([]*queuecall.Queuecall, error)
	UpdateStatusWaiting(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	Delete(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)

	Execute(ctx context.Context, queuecallID uuid.UUID, agentID uuid.UUID) (*queuecall.Queuecall, error)
	Kick(ctx context.Context, queuecallID uuid.UUID) (*queuecall.Queuecall, error)
	KickByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)

	HealthCheck(ctx context.Context, id uuid.UUID, retryCount int)

	EventCallCallHangup(ctx context.Context, referenceID uuid.UUID)
	EventCallConfbridgeJoined(ctx context.Context, referenceID uuid.UUID, confbridgeID uuid.UUID)
	EventCallConfbridgeLeaved(ctx context.Context, referenceID uuid.UUID, confbridgeID uuid.UUID)
	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error

	TimeoutService(ctx context.Context, queuecallID uuid.UUID)
	TimeoutWait(ctx context.Context, queuecallID uuid.UUID)

	ServiceStart(
		ctx context.Context,
		queueID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType queuecall.ReferenceType,
		referenceID uuid.UUID,
	) (*commonservice.Service, error)
}

// queuecallHandler define
type queuecallHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler

	queueHandler queuehandler.QueueHandler
}

// NewQueuecallHandler return AgentHandler interface
func NewQueuecallHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	queueHandler queuehandler.QueueHandler,
) QueuecallHandler {
	return &queuecallHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,

		queueHandler: queueHandler,
	}
}

// getDuration return the time duration between two timestamps
func getDuration(ctx context.Context, tmStart, tmEnd *time.Time) time.Duration {
	if tmStart == nil || tmEnd == nil {
		logrus.Errorf("Could not get duration. nil timestamp. tmStart: %v, tmEnd: %v", tmStart, tmEnd)
		return 0
	}

	return tmEnd.Sub(*tmStart)
}
