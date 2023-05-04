package groupcallhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// AnswerCall handles the answered call.
func (h *groupcallHandler) AnswerCall(ctx context.Context, id uuid.UUID, answerCallID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "AnswerCall",
		"groupcall_id":   id,
		"answer_call_id": answerCallID,
	})

	// update answer call id
	gc, err := h.UpdateAnswerCallID(ctx, id, answerCallID)
	if err != nil {
		log.Errorf("Could not update the answer call id. err: %v", err)
		return errors.Wrap(err, "Could not update the answer call id.")
	}

	_, err = h.answerCommon(ctx, gc)
	if err != nil {
		log.Errorf("Could not handle the answerCommon process correctly. err: %v", err)
		return errors.Wrap(err, "could not handle the answerCommon process correctly")
	}

	return nil
}

// AnswerGroupcall handles the answered groupcall id.
func (h *groupcallHandler) AnswerGroupcall(ctx context.Context, id uuid.UUID, answerGroupcallID uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "AnswerGroupcall",
		"groupcall_id":        id,
		"answer_groupcall_id": answerGroupcallID,
	})

	// update answer groupcall id
	gc, err := h.UpdateAnswerGroupcallID(ctx, id, answerGroupcallID)
	if err != nil {
		log.Errorf("Could not update the answer call id. err: %v", err)
		return nil, errors.Wrap(err, "Could not update the answer call id.")
	}

	res, err := h.answerCommon(ctx, gc)
	if err != nil {
		log.Errorf("Could not handle the answerCommon process correctly. err: %v", err)
		return nil, errors.Wrap(err, "could not handle the common answer")
	}

	return res, nil
}

// answerCommon handles common process for groupcall answered by groupcall/call.
func (h *groupcallHandler) answerCommon(ctx context.Context, gc *groupcall.Groupcall) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "answerCommon",
		"groupcall": gc,
	})

	// check the master groupcall id
	if gc.MasterGroupcallID != uuid.Nil {
		// we have master groupcall id. need to update the answer groupcall id of the master groupcall id
		log.Debugf("The groupcal has master groupcall id. Updating the answer groupcall id. master_groupcall_id: %s, answer_groupcall_id: %s", gc.MasterGroupcallID, gc.ID)
		_, err := h.reqHandler.CallV1GroupcallUpdateAnswerGroupcallID(ctx, gc.MasterCallID, gc.ID)
		if err != nil {
			log.Errorf("Could not update the answer groupcall id. err: %v", err)
			return nil, errors.Wrap(err, "could not update the answer groupcall id")
		}

		// we don't do anything more here.
		// because hangup others request will come up from the master groupcall's hangup request.
		return gc, nil
	}

	if gc.AnswerMethod != groupcall.AnswerMethodHangupOthers {
		log.Debugf("Unsupported answer method. answer_method: %s", gc.AnswerMethod)
		return nil, fmt.Errorf("unsupported answer method")
	}

	// hangup others
	if errHangup := h.HangingupOthers(ctx, gc); errHangup != nil {
		// could not hangup the other calls. but we don't do any other actions here.
		log.Errorf("Could not hang up the other calls correctly. err: %v", errHangup)
	}

	return gc, nil

}
