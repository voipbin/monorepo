package groupcallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// HangingupOthers hangs up the call except answered call.
func (h *groupcallHandler) HangingupOthers(ctx context.Context, gd *groupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "HangingupOthers",
		"groupcall": gd,
	})

	for _, callID := range gd.CallIDs {
		if callID == gd.AnswerCallID {
			continue
		}

		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.reqHandler.CallV1CallHangup(ctx, id)
		}(callID)
	}

	return nil
}

// Hangingup hangs up the groupcalls.
func (h *groupcallHandler) Hangingup(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Hangingup",
		"groupcall_id": id,
	})

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall. err: %v", err)
		return nil, errors.Wrap(err, "Could not get groupcall.")
	}

	for _, callID := range res.CallIDs {
		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.reqHandler.CallV1CallHangup(ctx, id)
		}(callID)
	}

	return res, nil
}
