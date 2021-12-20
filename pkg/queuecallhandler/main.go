package queuecallhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package queuecallhandler -destination ./mock_queuecallhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
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
		userID uint64,
		queueID uuid.UUID,
		referenceType queuecall.ReferenceType,
		referenceID uuid.UUID,
		exitActionID uuid.UUID,
		forwardActionID uuid.UUID,
		confbridgeID uuid.UUID,
		webhookURI string,
		webhookMethod string,
		source cmaddress.Address,
		routingMethod queue.RoutingMethod,
		tagIDs []uuid.UUID,
		timeoutWait int,
		timeoutService int,
	) (*queuecall.Queuecall, error)
	Execute(ctx context.Context, queuecallID uuid.UUID)
	Hangup(ctx context.Context, referenceID uuid.UUID)
	Kick(ctx context.Context, queuecallID uuid.UUID) error
	KickByReferenceID(ctx context.Context, referenceID uuid.UUID) error
	Leaved(ctx context.Context, referenceID, confbridgeID uuid.UUID)

	TimeoutService(ctx context.Context, queuecallID uuid.UUID)
	TimeoutWait(ctx context.Context, queuecallID uuid.UUID)
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

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
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
