package queuecallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package queuecallhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/models/service"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

const (
	defaultHealthCheckMaxRetryCount = 2
	defaultHealthCheckDelay         = 5000 // 5 seconds
)

// QueuecallHandler interface
type QueuecallHandler interface {
	Create(
		ctx context.Context,
		q *queue.Queue,
		referenceType queuecall.ReferenceType,
		referenceID uuid.UUID,
		referenceActiveflowID uuid.UUID,
		forwardActionID uuid.UUID,
		exitActionID uuid.UUID,
		conferenceID uuid.UUID,
		source commonaddress.Address,
	) (*queuecall.Queuecall, error)
	Get(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queuecall.Queuecall, error)
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
		exitActionID uuid.UUID,
	) (*service.Service, error)
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

// parseTime return the time.Time of parsed voipbin's timestamp string.
func parseTime(timestamp string) (time.Time, error) {
	layout := "2006-01-02 15:04:05.000000"

	if len(timestamp) < len(layout) {
		return time.Time{}, fmt.Errorf("wrong format")
	}
	ts := timestamp[:len(layout)]

	res, err := time.Parse(layout, ts)
	if err != nil {
		logrus.Errorf("Could not parse the timestamp. err: %v", err)
		return time.Time{}, err
	}

	return res, nil
}

// getDuration return the timeduration from the timestamp string
func getDuration(ctx context.Context, tmStart, tmEnd string) time.Duration {
	// get wait duration
	timeStart, err := parseTime(tmStart)
	if err != nil {
		logrus.Errorf("Could not parse the timestamp. err: %v", err)
		return 0
	}

	timeEnd, err := parseTime(tmEnd)
	if err != nil {
		logrus.Errorf("Could not parse the timestamp. err: %v", err)
		return 0
	}

	return timeEnd.Sub(timeStart)
}
