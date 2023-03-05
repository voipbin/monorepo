package groupcallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// AnswerGroupcall handles the answered group dial.
func (h *groupcallHandler) AnswerGroupcall(ctx context.Context, groupcallID uuid.UUID, answerCallID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AnswerGroupcall",
		"groupcall_id":   groupcallID,
		"answer_call_id": answerCallID,
	})

	// get groupcall
	gd, err := h.Get(ctx, groupcallID)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return errors.Wrap(err, "Could not get group dial info.")
	}

	if gd.AnswerMethod != groupcall.AnswerMethodHangupOthers {
		log.Debugf("Unsupported answer method. answer_method: %s", gd.AnswerMethod)
		return fmt.Errorf("unsupported answer method")
	}

	// update answer call id
	res, err := h.UpdateAnswerCallID(ctx, gd.ID, answerCallID)
	if err != nil {
		log.Errorf("Could not update the answer call id. err: %v", err)
		return errors.Wrap(err, "Could not update the answer call id.")
	}

	// hangup others
	if errHangup := h.HangupOthers(ctx, res); errHangup != nil {
		// could not hangup the other calls. but we don't do any other actions here.
		log.Errorf("Could not hang up the other calls correctly. err: %v", errHangup)
	}

	return nil
}
