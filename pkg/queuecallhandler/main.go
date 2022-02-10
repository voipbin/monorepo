package queuecallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package queuecallhandler -destination ./mock_queuecallhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
)

// List of default values
const (
	defaultDelayQueuecallExecute = 1000 // 1000 ms(1 sec)
)

// QueuecallHandler interface
type QueuecallHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		queueID uuid.UUID,
		referenceType queuecall.ReferenceType,
		referenceID uuid.UUID,
		flowID uuid.UUID,
		exitActionID uuid.UUID,
		forwardActionID uuid.UUID,
		confbridgeID uuid.UUID,
		source cmaddress.Address,
		routingMethod queue.RoutingMethod,
		tagIDs []uuid.UUID,
		timeoutWait int,
		timeoutService int,
	) (*queuecall.Queuecall, error)
	Execute(ctx context.Context, queuecallID uuid.UUID)
	Hungup(ctx context.Context, referenceID uuid.UUID)
	Kick(ctx context.Context, queuecallID uuid.UUID) (*queuecall.Queuecall, error)
	KickByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	Leaved(ctx context.Context, referenceID, confbridgeID uuid.UUID)
	Joined(ctx context.Context, referenceID, confbridgeID uuid.UUID)
	SearchAgent(ctx context.Context, queuecallID uuid.UUID)

	TimeoutService(ctx context.Context, queuecallID uuid.UUID)
	TimeoutWait(ctx context.Context, queuecallID uuid.UUID)

	Get(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error)
	GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queuecall.Queuecall, error)
}

// queuecallHandler define
type queuecallHandler struct {
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler

	queuecallReferenceHandler queuecallreferencehandler.QueuecallReferenceHandler
}

// NewQueuecallHandler return AgentHandler interface
func NewQueuecallHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	queuecallReferenceHandler queuecallreferencehandler.QueuecallReferenceHandler,
) QueuecallHandler {
	return &queuecallHandler{
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,

		queuecallReferenceHandler: queuecallReferenceHandler,
	}
}

// parseTime return the time.Time of parsed voipbin's timestamp string.
func parseTime(timestamp string) (time.Time, error) {
	layout := "2006-01-02 15:04:05.000000"
	tm, err := time.Parse(layout, timestamp)
	if err != nil {
		logrus.Errorf("Could not parse the timestamp. err: %v", err)
		return time.Time{}, err
	}

	return tm, nil
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
