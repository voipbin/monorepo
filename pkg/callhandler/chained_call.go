package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// ChainedCallIDAdd adds the chained call id to the call and set the added chained call's master call id.
func (h *callHandler) ChainedCallIDAdd(id, chainedCallID uuid.UUID) error {
	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"call":            id,
			"chained_call_id": chainedCallID,
		},
	)
	log.Debug("Adding the chained call id.")

	// transaction start
	tx, c, err := h.db.CallTXStart(id)
	if err != nil {
		log.Errorf("Could not start the transaction for chained call id add. err: %v", err)
		return err
	}

	// check the call's status
	switch c.Status {
	case call.StatusTerminating, call.StatusCanceling, call.StatusHangup:
		log.Errorf("The master call has invalid status. status: %s", c.Status)
		h.db.CallTXFinish(tx, false)
		return fmt.Errorf("the master call has invalid status. status: %s", c.Status)
	}

	// add the chained call id
	if err := h.db.CallTXAddChainedCallID(tx, id, chainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return err
	}

	// set the master call id
	if err := h.db.CallSetMasterCallID(ctx, chainedCallID, id); err != nil {
		log.Errorf("Could not set the chained call's master call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return err
	}

	// commit the changes
	h.db.CallTXFinish(tx, true)

	return nil
}

// ChainedCallIDRemove removes the chained call id and set the master call id to nil
func (h *callHandler) ChainedCallIDRemove(id, chainedCallID uuid.UUID) error {

	ctx := context.Background()

	log := logrus.WithFields(
		logrus.Fields{
			"call":            id,
			"chained_call_id": chainedCallID,
		},
	)
	log.Debug("Removing the chained call id.")

	// transaction start
	tx, c, err := h.db.CallTXStart(id)
	if err != nil {
		log.Errorf("Could not start the transaction for chained call id add. err: %v", err)
		return err
	}

	// check the call's status
	switch c.Status {
	case call.StatusTerminating, call.StatusCanceling, call.StatusHangup:
		log.Errorf("The master call has invalid status. status: %s", c.Status)
		h.db.CallTXFinish(tx, false)
		return fmt.Errorf("the master call has invalid status. status: %s", c.Status)
	}

	// remove the chained call id
	if err := h.db.CallTXRemoveChainedCallID(tx, id, chainedCallID); err != nil {
		log.Errorf("Could not add the chained call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return err
	}

	// set the master call id to nil
	if err := h.db.CallSetMasterCallID(ctx, chainedCallID, uuid.Nil); err != nil {
		log.Errorf("Could not set the chained call's master call id. err: %v", err)
		h.db.CallTXFinish(tx, false)
		return err
	}

	h.db.CallTXFinish(tx, true)
	return nil
}
