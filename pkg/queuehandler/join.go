package queuehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

// Join creates the new queuecall
func (h *queueHandler) Join(ctx context.Context, queueID uuid.UUID, referenceType queuecall.ReferenceType, referenceID uuid.UUID, exitActionID uuid.UUID) (*queuecall.Queuecall, error) {
	log := logrus.New().WithFields(
		logrus.Fields{
			"func":     "Join",
			"queue_id": queueID,
			"call_id":  referenceID,
		},
	)

	// get queue
	q, err := h.Get(ctx, queueID)
	if err != nil {
		log.Errorf("Could not get queue. err: %v", err)
		return nil, err
	}

	if referenceType != queuecall.ReferenceTypeCall {
		log.Errorf("unsupported reference type")
		return nil, fmt.Errorf("unsupported reference type")
	}

	// get source
	c, err := h.reqHandler.CMV1CallGet(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get reference info. err: %v", err)
		return nil, fmt.Errorf("reference info not found")
	}

	// get source
	var source cmaddress.Address
	if c.Direction == cmcall.DirectionIncoming {
		source = c.Source
	} else {
		source = c.Destination
	}

	// create a new queuecall
	res, err := h.queuecallHandler.Create(
		ctx,
		q.UserID,
		q.ID,
		referenceType,
		referenceID,
		q.ForwardActionID,
		exitActionID,
		q.ConfbridgeID,
		q.WebhookURI,
		q.WebhookMethod,
		source,
		q.RoutingMethod,
		q.TagIDs,
		q.WaitTimeout,
		q.ServiceTimeout,
	)
	if err != nil {
		log.Errorf("Could not create the queuecall. err: %v", err)
		return nil, err
	}

	// set the queuecallReference's current queuecall id.
	if err := h.queuecallReferenceHandler.SetCurrentQueuecallID(ctx, referenceID, referenceType, res.ID); err != nil {
		log.Errorf("Could not set the current queuecall id to the queuecallreference. err: %v", err)
		return nil, err
	}

	// add the queuecall to the queue.
	if err := h.db.QueueAddQueueCallID(ctx, res.QueueID, res.ID); err != nil {
		log.Errorf("Could not add the queuecall to the queue. err: %v", err)
	}

	return res, nil
}
