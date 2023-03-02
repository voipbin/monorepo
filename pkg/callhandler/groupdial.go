package callhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
)

// createGroupdial creates a new group dial.
func (h *callHandler) createGroupdial(
	ctx context.Context,
	customerID uuid.UUID,
	destination *commonaddress.Address,
	callIDs []uuid.UUID,
	ringMethod groupdial.RingMethod,
	answerMethod groupdial.AnswerMethod,
) (*groupdial.Groupdial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "createGroupdial",
		"customer_id":   customerID,
		"destination":   destination,
		"call_ids":      callIDs,
		"ring_method":   ringMethod,
		"answer_method": answerMethod,
	})

	id := h.utilHandler.CreateUUID()
	log = log.WithField("groupdial_id", id)

	// create group dial
	tmp := &groupdial.Groupdial{
		ID:         id,
		CustomerID: customerID,

		Destination:  destination,
		CallIDs:      callIDs,
		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,
	}

	if errCreate := h.db.GroupdialCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create the group dial. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "Could not create the group dial.")
	}

	res, err := h.db.GroupdialGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get created group dial info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupdial.EventTypeGroupdialCreated, res)
	log.WithField("groupdial", res).Debugf("Created a new groupdial. groupdial_id: %s", res.ID)

	return res, nil
}

// getGroupdial returns a groupdial of the given id.
func (h *callHandler) getGroupdial(ctx context.Context, id uuid.UUID) (*groupdial.Groupdial, error) {
	return h.db.GroupdialGet(ctx, id)
}

// updateGroupdialAnswerCallID updates the answer call id.
func (h *callHandler) updateGroupdialAnswerCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) (*groupdial.Groupdial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "updateGroupdialAnswerCallID",
		"groupdial_id": id,
		"call_id":      callID,
	})

	gd, err := h.getGroupdial(ctx, id)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get group dial info.")
	}

	gd.AnswerCallID = callID
	if errUpdate := h.db.GroupdialUpdate(ctx, gd); errUpdate != nil {
		log.Errorf("Could not update the group dial info. err: %v", errUpdate)
		return nil, errors.Wrap(errUpdate, "Could not update the group dial info.")
	}

	res, err := h.db.GroupdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated group dial info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get updated group dial info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupdial.EventTypeGroupdialAnswered, res)

	return res, nil
}

// answerGroupdial handles the answered group dial.
func (h *callHandler) answerGroupdial(ctx context.Context, groupdialID uuid.UUID, answercallID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "answerGroupdial",
		"groupdial_id":   groupdialID,
		"answer_call_id": answercallID,
	})

	// get groupdial
	gd, err := h.getGroupdial(ctx, groupdialID)
	if err != nil {
		log.Errorf("Could not get group dial info. err: %v", err)
		return errors.Wrap(err, "Could not get group dial info.")
	}

	if gd.AnswerMethod != groupdial.AnswerMethodHangupOthers {
		log.Debugf("Unsupported answer method. answer_method: %s", gd.AnswerMethod)
		return fmt.Errorf("unsupported answer method")
	}

	// update answer call id
	tmp, err := h.updateGroupdialAnswerCallID(ctx, gd.ID, answercallID)
	if err != nil {
		log.Errorf("Could not update the answer call id. err: %v", err)
		return errors.Wrap(err, "Could not update the answer call id.")
	}

	for _, callID := range tmp.CallIDs {
		if callID == answercallID {
			continue
		}

		log.Debugf("Hanging up the groupdial calls. call_id: %s", callID)
		go func(id uuid.UUID) {
			_, _ = h.HangingUp(ctx, id, call.HangupReasonNormal)
		}(callID)
	}

	return nil
}
