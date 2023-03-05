package groupcallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// HangupOthers hangs up the call except answered call.
func (h *groupcallHandler) HangupOthers(ctx context.Context, gd *groupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "HangupOthers",
		"groupcall": gd,
	})

	for _, callID := range gd.CallIDs {
		if callID == gd.AnswerCallID {
			continue
		}

		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.callHandler.HangingUp(ctx, id, call.HangupReasonNormal)
		}(callID)
	}

	return nil
}

// Hangup hangs up the groupcalls.
func (h *groupcallHandler) Hangup(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Hangup",
		"groupcall_id": id,
	})

	gd, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall. err: %v", err)
		return errors.Wrap(err, "Could not get groupcall.")
	}

	for _, callID := range gd.CallIDs {
		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.callHandler.HangingUp(ctx, id, call.HangupReasonNormal)
		}(callID)
	}

	return nil
}
