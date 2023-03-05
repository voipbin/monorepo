package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// createGroupcall creates a new group dial.
func (h *callHandler) createGroupcall(
	ctx context.Context,
	customerID uuid.UUID,
	destination *commonaddress.Address,
	callIDs []uuid.UUID,
	ringMethod groupcall.RingMethod,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "createGroupcall",
		"customer_id":   customerID,
		"destination":   destination,
		"call_ids":      callIDs,
		"ring_method":   ringMethod,
		"answer_method": answerMethod,
	})

	id := h.utilHandler.CreateUUID()
	log = log.WithField("groupcall_id", id)

	// create group dial
	tmp := &groupcall.Groupcall{
		ID:         id,
		CustomerID: customerID,

		Destinations: []commonaddress.Address{*destination},
		CallIDs:      callIDs,
		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,
	}

	if errCreate := h.db.GroupcallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create the group dial. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "Could not create the group dial.")
	}

	res, err := h.db.GroupcallGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get created group dial info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallCreated, res)
	log.WithField("groupcall", res).Debugf("Created a new groupcall. groupcall_id: %s", res.ID)

	return res, nil
}

// getGroupcall returns a groupcall of the given id.
func (h *callHandler) getGroupcall(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	return h.db.GroupcallGet(ctx, id)
}

// updateGroupcallAnswerCallID updates the answer call id.
func (h *callHandler) updateGroupcallAnswerCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "updateGroupcallAnswerCallID",
		"groupcall_id": id,
		"call_id":      callID,
	})

	gd, err := h.getGroupcall(ctx, id)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get group dial info.")
	}

	gd.AnswerCallID = callID
	if errUpdate := h.db.GroupcallUpdate(ctx, gd); errUpdate != nil {
		log.Errorf("Could not update the group dial info. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "Could not update the group dial info.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get updated group dial info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallAnswered, res)

	return res, nil
}

// answerGroupcall handles the answered group dial.
func (h *callHandler) answerGroupcall(ctx context.Context, groupcallID uuid.UUID, answercallID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "answerGroupcall",
		"groupcall_id":   groupcallID,
		"answer_call_id": answercallID,
	})

	// get groupcall
	gd, err := h.getGroupcall(ctx, groupcallID)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return errors.Wrap(err, "Could not get group dial info.")
	}

	if gd.AnswerMethod != groupcall.AnswerMethodHangupOthers {
		log.Debugf("Unsupported answer method. answer_method: %s", gd.AnswerMethod)
		return fmt.Errorf("unsupported answer method")
	}

	// update answer call id
	tmp, err := h.updateGroupcallAnswerCallID(ctx, gd.ID, answercallID)
	if err != nil {
		log.Errorf("Could not update the answer call id. err: %v", err)
		return errors.Wrap(err, "Could not update the answer call id.")
	}

	for _, callID := range tmp.CallIDs {
		if callID == answercallID {
			continue
		}

		log.Debugf("Hanging up the groupcall calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.HangingUp(ctx, id, call.HangupReasonNormal)
		}(callID)
	}

	return nil
}
