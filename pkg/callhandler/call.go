package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

// Delete deletes the call
func (h *callHandler) Delete(ctx context.Context, id uuid.UUID) (*call.Call, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Delete",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return nil, err
	}

	if c.TMDelete != dbhandler.DefaultTimeStamp {
		// the call has been deleted already.
		return c, nil
	}

	if c.Status != call.StatusHangup {
		// hangup the call
		tmp, err := h.HangingUp(ctx, id, call.HangupReasonNormal)
		if err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
			return nil, err
		}
		log.WithField("call", tmp).Debugf("The call is on progressing. Hanging up the call. call_id: %s", tmp.ID)
	}

	// delete the call
	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the call. err: %v", err)
		return nil, err
	}

	return res, nil
}
