package groupcallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// Create creates a new groupcall.
func (h *groupcallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	callIDs []uuid.UUID,
	ringMethod groupcall.RingMethod,
	answerMethod groupcall.AnswerMethod,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Create",
		"customer_id":   customerID,
		"source":        source,
		"destinations":  destinations,
		"call_ids":      callIDs,
		"ring_method":   ringMethod,
		"answer_method": answerMethod,
	})

	id := h.utilHandler.CreateUUID()
	log = log.WithField("groupcall_id", id)

	// create groupcall
	tmp := &groupcall.Groupcall{
		ID:         id,
		CustomerID: customerID,

		Source:       source,
		Destinations: destinations,

		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,

		AnswerCallID: uuid.Nil,
		CallIDs:      callIDs,

		CallCount: len(callIDs),
	}

	if errCreate := h.db.GroupcallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create the groupcall. err: %v", errCreate)
		return nil, errors.Wrap(errCreate, "Could not create the groupcall.")
	}

	res, err := h.db.GroupcallGet(ctx, tmp.ID)
	if err != nil {
		log.Errorf("Could not get created groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get created groupcall info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallCreated, res)
	log.WithField("groupcall", res).Debugf("Created a new groupcall. groupcall_id: %s", res.ID)

	return res, nil
}

// Get returns a groupcall of the given id.
func (h *groupcallHandler) Get(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get groupcall.")
	}

	return res, nil
}

// Gets returns list of groupcalls.
func (h *groupcallHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
	})

	res, err := h.db.GroupcallGets(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get calls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateAnswerCallID updates the answer call id.
func (h *groupcallHandler) UpdateAnswerCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateAnswerCallID",
		"groupcall_id": id,
		"call_id":      callID,
	})

	if errSet := h.db.GroupcallSetAnswerCallID(ctx, id, callID); errSet != nil {
		log.Errorf("Could not set the answer call id. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not set answer call id.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get updated groupcall info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallProgressing, res)

	return res, nil
}

// Delete deletes the groupcall.
func (h *groupcallHandler) Delete(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "Delete",
		"groupcall_id": id,
	})

	if errSet := h.db.GroupcallDelete(ctx, id); errSet != nil {
		log.Errorf("Could not delete the groupcall. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not delete the groupcall id.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get deleted groupcall info.")
	}
	h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallDeleted, res)

	return res, nil
}

// DecreaseCallCount decreases the groupcall's call count.
func (h *groupcallHandler) DecreaseCallCount(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "DecreaseCallCount",
		"groupcall_id": id,
	})

	if errDecrease := h.db.GroupcallDecreaseCallCount(ctx, id); errDecrease != nil {
		log.Errorf("Could not decrease the groupcall call_count. err: %v", errDecrease)
		return nil, errors.Wrap(errDecrease, "Could not decrease the groupcall call_count.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get decreased groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get decreased groupcall info.")
	}

	if res.CallCount <= 0 {
		log.Debugf("Groupcall's call count is 0. groupcall_id: %s", res.ID)
		h.notifyHandler.PublishEvent(ctx, groupcall.EventTypeGroupcallHangup, res)
	}

	return res, nil
}
