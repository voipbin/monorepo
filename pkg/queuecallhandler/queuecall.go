package queuecallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// Gets returns queuecalls
func (h *queuecallHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*queuecall.Queuecall, error) {
	log := logrus.WithField("func", "Gets")

	res, err := h.db.QueuecallGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get queuecalls info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns queuecall info.
func (h *queuecallHandler) Get(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithField("func", "Get")

	res, err := h.db.QueuecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get queuecall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID returns queuecall info of the given referenceID.
func (h *queuecallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "GetByReferenceID",
			"reference_id": referenceID,
		})

	qcf, err := h.queuecallReferenceHandler.Get(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get queuecall reference. err: %v", err)
		return nil, err
	}

	if qcf.CurrentQueuecallID == uuid.Nil {
		log.Errorf("No current queuecall info exist.")
		return nil, fmt.Errorf("no current queuecall id info")
	}

	// get current queuecall info
	res, err := h.db.QueuecallGet(ctx, qcf.CurrentQueuecallID)
	if err != nil {
		log.Errorf("Could not get queuecall reference info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// QueueCreate creates a new queue.
func (h *queuecallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	queueID uuid.UUID,
	referenceType queuecall.ReferenceType,
	referenceID uuid.UUID,
	flowID uuid.UUID,
	forwardActionID uuid.UUID,
	exitActionID uuid.UUID,
	confbridgeID uuid.UUID,
	source cmaddress.Address,
	routingMethod queue.RoutingMethod,
	tagIDs []uuid.UUID,
	timeoutWait int,
	timeoutService int,
) (*queuecall.Queuecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"queue_id":       queueID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})
	log.Debug("Creating a new queuecall.")

	// generate queue id
	id := uuid.Must(uuid.NewV4())
	log = log.WithField("queuecall_id", id)

	c := &queuecall.Queuecall{
		ID:            id,
		CustomerID:    customerID,
		QueueID:       queueID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		FlowID:          flowID,
		ForwardActionID: forwardActionID,
		ExitActionID:    exitActionID,
		ConfbridgeID:    confbridgeID,

		Source:        source,
		RoutingMethod: routingMethod,
		TagIDs:        tagIDs,

		Status:         queuecall.StatusWait,
		ServiceAgentID: uuid.Nil,

		TimeoutWait:    timeoutWait,
		TimeoutService: timeoutService,

		TMCreate:  dbhandler.GetCurTime(),
		TMService: dbhandler.DefaultTimeStamp,
		TMUpdate:  dbhandler.DefaultTimeStamp,
		TMDelete:  dbhandler.DefaultTimeStamp,
	}

	// create
	if err := h.db.QueuecallCreate(ctx, c); err != nil {
		log.Errorf("Could not create a new queuecall. err: %v", err)
		return nil, err
	}

	// get created queuecall and notify
	res, err := h.db.QueuecallGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get created queuecall. err: %v", err)
		return nil, err
	}
	h.notifyhandler.PublishWebhookEvent(ctx, res.CustomerID, queuecall.EventTypeQueuecallCreated, res)

	// send the queuecall timeout-wait
	if res.TimeoutWait > 0 {
		if errTiemout := h.reqHandler.QMV1QueuecallTimeoutWait(ctx, res.ID, res.TimeoutWait); errTiemout != nil {
			log.Errorf("Could not send the timeout-wait request. err: %v", errTiemout)
		}
	}

	return res, nil
}
