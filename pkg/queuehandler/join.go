package queuehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/ttacon/libphonenumber"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
)

const (
	defaultSourceType   = cmaddress.TypeTel
	defaultSourceTarget = "+821021656521"
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

	// check the reference type. currently support the call type only.
	if referenceType != queuecall.ReferenceTypeCall {
		log.Errorf("unsupported reference type")
		return nil, fmt.Errorf("unsupported reference type")
	}

	// get call
	c, err := h.reqHandler.CMV1CallGet(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get reference info. err: %v", err)
		return nil, fmt.Errorf("reference info not found")
	}
	log.WithField("call", c).Debug("Found call info.")

	// get source
	source := h.getSource(c)
	log.WithField("source", source).Debug("Source address info.")

	// create confbridge
	cb, err := h.reqHandler.CMV1ConfbridgeCreate(ctx)
	if err != nil {
		log.Errorf("Could not create the confbridge. err: %v", err)
		return nil, err
	}

	// create queue flow
	f, err := h.createQueueFlow(ctx, q.CustomerID, q.ID, cb.ID, q.WaitActions)
	if err != nil {
		log.Errorf("Could not create the queue flow. err: %v", err)
		return nil, err
	}

	// get flow target action id
	forwardActionID, err := h.getForwardActionID(ctx, f)
	if err != nil {
		log.Errorf("Could not get forward action id. err: %v", err)
		return nil, err
	}

	// create a new queuecall
	res, err := h.queuecallHandler.Create(
		ctx,
		q.CustomerID,
		q.ID,
		referenceType,
		referenceID,
		f.ID,
		forwardActionID,
		exitActionID,
		cb.ID,
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

// getSource returns cmaddress for source.
func (h *queueHandler) getSource(c *cmcall.Call) cmaddress.Address {
	log := logrus.WithField("call_id", c.ID)

	var res cmaddress.Address
	if c.Direction == cmcall.DirectionIncoming {
		res = c.Source
	} else {
		res = c.Destination
	}

	valid := true
	if res.Type == cmaddress.TypeTel {
		num, err := libphonenumber.Parse(res.Target, "US")
		if err != nil {
			log.Debugf("Could not parse the number. err: %v", err)
			valid = false
		}

		if valid && !libphonenumber.IsValidNumber(num) {
			log.Debugf("Given source target is not valid. num: %v", num)
			valid = false
		}

	} else {
		valid = false
	}

	// check the address is valid or not.
	if !valid {
		res.Type = defaultSourceType
		res.Target = defaultSourceTarget
	}

	return res
}
