package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// ChainedCallIDAdd adds the chained call id to the call and set the added chained call's master call id.
func (h *callHandler) ChainedCallIDAdd(id, chainedCallID uuid.UUID) error {

	ctx := context.Background()

	// add the chanied call id
	if err := h.db.CallAddChainedCallID(ctx, id, chainedCallID); err != nil {
		logrus.Errorf("Could not add the chained call id. err: %v", err)
		return err
	}

	// set the master call id
	if err := h.db.CallSetMasterCallID(ctx, chainedCallID, id); err != nil {
		logrus.Errorf("Could not set the chained call's master call id. err: %v", err)
		return err
	}

	return nil
}

// ChainedCallIDRemove removes the chained call id and set the master call id to nil
func (h *callHandler) ChainedCallIDRemove(id, chainedCallID uuid.UUID) error {

	ctx := context.Background()

	// add the chanied call id
	if err := h.db.CallRemoveChainedCallID(ctx, id, chainedCallID); err != nil {
		logrus.Errorf("Could not remove the chained call id. err: %v", err)
		return err
	}

	// set the master call id to nil
	if err := h.db.CallSetMasterCallID(ctx, chainedCallID, uuid.Nil); err != nil {
		logrus.Errorf("Could not set the chained call's master call id to nil. err: %v", err)
		return err
	}

	return nil
}
