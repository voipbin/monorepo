package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/call"
)

// ChainedCallIDAdd adds the chained call id to the call and set the added chained call's master call id.
func (h *callHandler) ChainedCallIDAdd(ctx context.Context, id uuid.UUID, chainedCallID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ChainedCallIDAdd",
		"call":            id,
		"chained_call_id": chainedCallID,
	})
	log.Debug("Adding the chained call id.")

	// transaction start
	tx, c, err := h.db.CallTXStart(id)
	if err != nil {
		log.Errorf("Could not start the transaction for chained call id add. err: %v", err)
		return nil, err
	}

	// check the call's status
	switch c.Status {
	case call.StatusTerminating, call.StatusCanceling, call.StatusHangup:
		log.Errorf("The master call has invalid status. status: %s", c.Status)
		h.db.CallTXFinish(tx, false)
		return nil, fmt.Errorf("the master call has invalid status. status: %s", c.Status)
	}

	// add the chained call id
	if err := h.db.CallTXAddChainedCallID(tx, id, chainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return nil, err
	}

	// set the master call id
	if err := h.db.CallSetMasterCallID(ctx, chainedCallID, id); err != nil {
		log.Errorf("Could not set the chained call's master call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return nil, err
	}

	// commit the changes
	h.db.CallTXFinish(tx, true)

	masterCall, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated master call info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, masterCall.CustomerID, call.EventTypeCallUpdated, masterCall)

	chainedCall, err := h.db.CallGet(ctx, chainedCallID)
	if err != nil {
		log.Errorf("Could not get updated chained call info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, chainedCall.CustomerID, call.EventTypeCallUpdated, chainedCall)

	return masterCall, nil
}

// ChainedCallIDRemove removes the chained call id and set the master call id to nil
func (h *callHandler) ChainedCallIDRemove(ctx context.Context, id uuid.UUID, chainedCallID uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ChainedCallIDRemove",
		"call":            id,
		"chained_call_id": chainedCallID,
	})
	log.Debug("Removing the chained call id.")

	// transaction start
	tx, c, err := h.db.CallTXStart(id)
	if err != nil {
		log.Errorf("Could not start the transaction for chained call id add. err: %v", err)
		return nil, err
	}

	// check the call's status
	switch c.Status {
	case call.StatusTerminating, call.StatusCanceling, call.StatusHangup:
		log.Errorf("The master call has invalid status. status: %s", c.Status)
		h.db.CallTXFinish(tx, false)
		return nil, fmt.Errorf("the master call has invalid status. status: %s", c.Status)
	}

	// remove the chained call id
	if err := h.db.CallTXRemoveChainedCallID(tx, id, chainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return nil, err
	}

	// set the master call id to nil
	if err := h.db.CallSetMasterCallID(ctx, chainedCallID, uuid.Nil); err != nil {
		log.Errorf("Could not set the chained call's master call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return nil, err
	}

	h.db.CallTXFinish(tx, true)

	masterCall, err := h.db.CallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated master call info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, masterCall.CustomerID, call.EventTypeCallUpdated, masterCall)

	chainedCall, err := h.db.CallGet(ctx, chainedCallID)
	if err != nil {
		log.Errorf("Could not get updated chained call info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, chainedCall.CustomerID, call.EventTypeCallUpdated, chainedCall)

	return masterCall, nil
}
