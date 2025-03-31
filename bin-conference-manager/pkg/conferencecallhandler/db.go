package conferencecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-conference-manager/models/conferencecall"
)

// Create is handy function for creating a conference.
// it increases corresponded counter
func (h *conferencecallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	activeflowID uuid.UUID,
	conferenceID uuid.UUID,
	referenceType conferencecall.ReferenceType,
	referenceID uuid.UUID,
) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"customer_id":    customerID,
		"activeflow_id":  activeflowID,
		"conference_id":  conferenceID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	id := h.utilHandler.UUIDCreate()
	tmp := &conferencecall.Conferencecall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		ActiveflowID: activeflowID,
		ConferenceID: conferenceID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Status: conferencecall.StatusJoining,
	}
	log = log.WithField("conferencecall_id", id.String())
	log.WithField("conferencecall", tmp).Debugf("Creating conferencecall. conference_call: %s", tmp.ID)

	if errCreate := h.db.ConferencecallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new conferencecall. err: %v", errCreate)
		return nil, errCreate
	}
	promConferencecallTotal.WithLabelValues(string(tmp.ReferenceType), string(conferencecall.StatusJoining)).Inc()

	res, err := h.db.ConferencecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created conferencecall info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, conferencecall.EventTypeConferencecallJoining, res)

	// start health check
	go func() {
		_ = h.reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, id, 0, defaultHealthCheckDelay)
	}()

	return res, nil
}

// Gets returns list of conferencecalls.
func (h *conferencecallHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conferencecall.Conferencecall, error) {

	res, err := h.db.ConferencecallGets(ctx, size, token, filters)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Get is handy function for getting a conferencecall.
func (h *conferencecallHandler) Get(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Get",
		"conferencecall_id": id,
	})

	res, err := h.db.ConferencecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID is handy function for getting a conferencecall by the reference_id.
func (h *conferencecallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error) {
	res, err := h.db.ConferencecallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get conferencecall info.")
	}

	return res, nil
}

// updateStatus is handy function for update the conferencecall's status.
// it increases corresponded counter
func (h *conferencecallHandler) updateStatus(ctx context.Context, id uuid.UUID, status conferencecall.Status) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "updateStatus",
		"conferencecall_id": id,
		"status":            status,
	})

	if errStatus := h.db.ConferencecallUpdateStatus(ctx, id, status); errStatus != nil {
		log.Errorf("Could not update the conferencecall status. err: %v", errStatus)
		return nil, errStatus
	}

	// get updated conferencecall
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated conferencecall. err: %v", err)
		return nil, err
	}
	promConferencecallTotal.WithLabelValues(string(res.ReferenceType), string(res.Status)).Inc()

	return res, nil
}

// updateStatusJoined is handy function for update the conferencecall's status to the joined.
// it increases corresponded counter
func (h *conferencecallHandler) updateStatusJoined(ctx context.Context, conferencecallID uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "updateStatusJoined",
		"conferencecall_id": conferencecallID,
	})

	res, err := h.updateStatus(ctx, conferencecallID, conferencecall.StatusJoined)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, conferencecall.EventTypeConferencecallJoined, res)

	return res, nil
}

// updateStatusLeaving is handy function for update the conferencecall's status to the leaved.
// it increases corresponded counter
func (h *conferencecallHandler) updateStatusLeaving(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "updateStatusLeaving",
		"conferencecall_id": id,
	})

	res, err := h.updateStatus(ctx, id, conferencecall.StatusLeaving)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, conferencecall.EventTypeConferencecallLeaving, res)

	return res, nil
}

// updateStatusLeaved is handy function for update the conferencecall's status to the leaved.
// it increases corresponded counter
func (h *conferencecallHandler) updateStatusLeaved(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "updateStatusLeaved",
		"conferencecall_id": id,
	})

	res, err := h.updateStatus(ctx, id, conferencecall.StatusLeaved)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, conferencecall.EventTypeConferencecallLeaved, res)

	return res, nil
}
