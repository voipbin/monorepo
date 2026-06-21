package groupcallhandler

import (
	"context"
	stderrors "errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// Create creates a new groupcall.
func (h *groupcallHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	ownerType commonidentity.OwnerType,
	ownerID uuid.UUID,
	flowID uuid.UUID,
	source *commonaddress.Address,
	destinations []commonaddress.Address,
	callIDs []uuid.UUID,
	groupcallIDs []uuid.UUID,
	masterCallID uuid.UUID,
	masterGroupcallID uuid.UUID,
	ringMethod groupcall.RingMethod,
	answerMethod groupcall.AnswerMethod,
	anonymous string,
) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"id":             id,
		"customer_id":    customerID,
		"owner_type":     ownerType,
		"owner_id":       ownerID,
		"source":         source,
		"destinations":   destinations,
		"call_ids":       callIDs,
		"groupcall_ids":  groupcallIDs,
		"master_call_id": masterCallID,
		"ring_method":    ringMethod,
		"answer_method":  answerMethod,
		"anonymous":      anonymous,
	})

	// Canonicalize the stored source/destinations through the shared address
	// normalization authority. NormalizeTarget is loss-proof, so the error is
	// safely discarded. Nil-guard the source pointer; normalize destinations by
	// index (range copy would discard the result).
	if source != nil {
		source.Target, _ = commonaddress.NormalizeTarget(source.Type, source.Target)
	}
	for i := range destinations {
		destinations[i].Target, _ = commonaddress.NormalizeTarget(destinations[i].Type, destinations[i].Target)
	}

	// create groupcall
	tmp := &groupcall.Groupcall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: ownerType,
			OwnerID:   ownerID,
		},

		Status: groupcall.StatusProgressing,
		FlowID: flowID,

		Source:       source,
		Destinations: destinations,

		MasterCallID:      masterCallID,
		MasterGroupcallID: masterGroupcallID,

		RingMethod:   ringMethod,
		AnswerMethod: answerMethod,
		Anonymous:    anonymous,

		AnswerCallID: uuid.Nil,
		CallIDs:      callIDs,

		AnswerGroupcallID: uuid.Nil,
		GroupcallIDs:      groupcallIDs,

		CallCount:      len(callIDs),
		GroupcallCount: len(groupcallIDs),
		DialIndex:      0,
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, groupcall.EventTypeGroupcallCreated, res)
	log.WithField("groupcall", res).Debugf("Created a new groupcall. groupcall_id: %s", res.ID)

	return res, nil
}

// Get returns a groupcall of the given id.
//
// When the underlying DB layer returns dbhandler.ErrNotFound, Get returns a
// typed *cerrors.VoipbinError (Status=NotFound) so the api-manager edge can
// recover the upstream domain/reason via errors.As.
func (h *groupcallHandler) Get(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameCallManager,
				"GROUPCALL_NOT_FOUND",
				"The group call was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrap(err, "Could not get groupcall.")
	}

	return res, nil
}

// List returns list of groupcalls.
func (h *groupcallHandler) List(ctx context.Context, size uint64, token string, filters map[groupcall.Field]any) ([]*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "List",
		"filters": filters,
	})

	res, err := h.db.GroupcallList(ctx, size, token, filters)
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, groupcall.EventTypeGroupcallProgressing, res)

	return res, nil
}

// UpdateAnswerGroupcallID updates the answer groupcall id.
func (h *groupcallHandler) UpdateAnswerGroupcallID(ctx context.Context, id uuid.UUID, answerGroupcallID uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                "UpdateAnswerGroupcallID",
		"groupcall_id":        id,
		"answer_groupcall_id": answerGroupcallID,
	})

	if errSet := h.db.GroupcallSetAnswerGroupcallID(ctx, id, answerGroupcallID); errSet != nil {
		log.Errorf("Could not set the answer groupcall id. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not set answer groupcall id.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get updated groupcall info.")
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, groupcall.EventTypeGroupcallProgressing, res)

	return res, nil
}

// dbDelete deletes the groupcall.
func (h *groupcallHandler) dbDelete(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "dbDelete",
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, groupcall.EventTypeGroupcallDeleted, res)

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

	return res, nil
}

// DecreaseGroupcallCount decreases the groupcall's groupcall count.
func (h *groupcallHandler) DecreaseGroupcallCount(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "DecreaseGroupcallCount",
		"groupcall_id": id,
	})

	if errDecrease := h.db.GroupcallDecreaseGroupcallCount(ctx, id); errDecrease != nil {
		log.Errorf("Could not decrease the groupcall groupcall_count. err: %v", errDecrease)
		return nil, errors.Wrap(errDecrease, "Could not decrease the groupcall groupcall_count.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get decreased groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get decreased groupcall info.")
	}

	return res, nil
}

// UpdateCallIDsAndCallCountAndDialIndex updates the given groupcall's call_ids, call_count, dial_index.
func (h *groupcallHandler) UpdateCallIDsAndCallCountAndDialIndex(ctx context.Context, id uuid.UUID, callIDs []uuid.UUID, callCount int, dialIndex int) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateCallIDsAndCallCountAndDialIndex",
		"groupcall_id": id,
		"call_ids":     callIDs,
		"call_count":   callCount,
		"dial_index":   dialIndex,
	})

	if errSet := h.db.GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, id, callIDs, callCount, dialIndex); errSet != nil {
		log.Errorf("Could not decrease the groupcall call_count. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not decrease the groupcall call_count.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get decreased groupcall info.")
	}

	return res, nil
}

// UpdateGroupcallIDsAndGroupcallCountAndDialIndex updates the given groupcall's groupcall_ids, groupcall_count, dial_index.
func (h *groupcallHandler) UpdateGroupcallIDsAndGroupcallCountAndDialIndex(ctx context.Context, id uuid.UUID, groupcallIDs []uuid.UUID, groupcallCount int, dialIndex int) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "UpdateGroupcallIDsAndGroupcallCountAndDialIndex",
		"groupcall_id":    id,
		"groupcall_ids":   groupcallIDs,
		"groupcall_count": groupcallCount,
		"dial_index":      dialIndex,
	})

	if errSet := h.db.GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, id, groupcallIDs, groupcallCount, dialIndex); errSet != nil {
		log.Errorf("Could not update the groupcall info. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not update the groupcall info.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get decreased groupcall info.")
	}

	return res, nil
}

// UpdateStatus updates the given groupcall's status.
func (h *groupcallHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status groupcall.Status) (*groupcall.Groupcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "UpdateStatus",
		"groupcall_id": id,
		"status":       status,
	})

	if errSet := h.db.GroupcallSetStatus(ctx, id, status); errSet != nil {
		log.Errorf("Could not decrease the groupcall call_count. err: %v", errSet)
		return nil, errors.Wrap(errSet, "Could not decrease the groupcall call_count.")
	}

	res, err := h.db.GroupcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated groupcall info. err: %v", err)
		return nil, errors.Wrap(err, "Could not get decreased groupcall info.")
	}

	return res, nil
}
