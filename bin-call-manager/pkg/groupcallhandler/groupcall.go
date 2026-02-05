package groupcallhandler

import (
	"context"

	"monorepo/bin-call-manager/models/groupcall"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Delete deletes the groupcall.
func (h *groupcallHandler) Delete(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Delete",
		"groupcall_id": id,
	})

	// get
	gc, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
		return nil, err
	}

	if gc.TMDelete != nil {
		// the call has been deleted already.
		return gc, nil
	}

	if gc.Status != groupcall.StatusHangup {
		// hangup the groupcall
		tmp, err := h.Hangingup(ctx, id)
		if err != nil {
			log.Errorf("Could not hangup the groupcall. err: %v", err)
			return nil, err
		}
		log.WithField("groupcall", tmp).Debugf("The groupcall is on progressing. Hanging up the groupcall. groupcall_id: %s", tmp.ID)
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete groupcall. err: %v", err)
		return nil, err
	}

	return res, nil
}
