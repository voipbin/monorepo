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

// Get returns a groupcall of the given id.
func (h *groupcallHandler) Get(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	return h.db.GroupcallGet(ctx, id)
}

// UpdateAnswerCallID updates the answer call id.
func (h *groupcallHandler) UpdateAnswerCallID(ctx context.Context, id uuid.UUID, callID uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateAnswerCallID",
		"groupcall_id": id,
		"call_id":      callID,
	})

	gd, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get groupcall info. err: %v", err)
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
