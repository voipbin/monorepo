package queuecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

// QueueCreate creates a new queue.
func (h *queuecallHandler) Create(
	ctx context.Context,
	userID uint64,
	queueID uuid.UUID,
	referenceType queuecall.ReferenceType,
	referenceID uuid.UUID,
	flowID uuid.UUID,
	forwardActionID uuid.UUID,
	exitActionID uuid.UUID,
	confbridgeID uuid.UUID,
	webhookURI string,
	webhookMethod string,
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
		UserID:        userID,
		QueueID:       queueID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		FlowID:          flowID,
		ForwardActionID: forwardActionID,
		ExitActionID:    exitActionID,
		ConfbridgeID:    confbridgeID,

		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,

		Source:        source,
		RoutingMethod: routingMethod,
		TagIDs:        tagIDs,

		Status:         queuecall.StatusWait,
		ServiceAgentID: uuid.Nil,

		TimeoutWait:    timeoutWait,
		TimeoutService: timeoutService,

		TMCreate:  getCurTime(),
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
	h.notifyhandler.PublishWebhookEvent(ctx, queuecall.EventTypeQueuecallCreated, c.WebhookURI, res)

	return res, nil
}
