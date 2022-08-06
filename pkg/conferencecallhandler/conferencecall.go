package conferencecallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
)

// Create is handy function for creating a conference.
// it increases corresponded counter
func (h *conferencecallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	conferenceID uuid.UUID,
	referenceType conferencecall.ReferenceType,
	referenceID uuid.UUID,
) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "Create",
			"customer_id":   customerID,
			"conference_id": conferenceID,
		},
	)

	id := uuid.Must(uuid.NewV4())
	log = log.WithField("conferencecall_id", id.String())

	curTime := dbhandler.GetCurTime()
	tmp := &conferencecall.Conferencecall{
		ID:           id,
		CustomerID:   customerID,
		ConferenceID: conferenceID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Status: conferencecall.StatusJoining,

		TMCreate: curTime,
		TMUpdate: curTime,
		TMDelete: dbhandler.DefaultTimeStamp,
	}

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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, conferencecall.EventTypeConferencecallJoining, res)

	return res, nil
}

// Get is handy function for getting a conferencecall.
func (h *conferencecallHandler) Get(
	ctx context.Context,
	id uuid.UUID,
) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "Get",
			"conferencecall_id": id,
		},
	)

	res, err := h.db.ConferencecallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID is handy function for getting a conferencecall by the reference_id.
func (h *conferencecallHandler) GetByReferenceID(
	ctx context.Context,
	referenceID uuid.UUID,
) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "GetByReferenceID",
			"reference_id": referenceID,
		},
	)

	res, err := h.db.ConferencecallGetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// updateStatus is handy function for update the conferencecall's status.
// it increases corresponded counter
func (h *conferencecallHandler) updateStatus(ctx context.Context, id uuid.UUID, status conferencecall.Status) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "updateStatus",
			"conferencecall_id": id,
			"status":            status,
		},
	)

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

// updateStatusByReferenceID is handy function for update the conferencecall's status.
// it increases corresponded counter
func (h *conferencecallHandler) updateStatusByReferenceID(
	ctx context.Context,
	referenceID uuid.UUID,
	status conferencecall.Status,
) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":         "updateStatusByReferenceID",
			"reference_id": referenceID,
			"status":       status,
		},
	)

	// get conferencecall
	cc, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get conferencecall info. err: %v", err)
		return nil, err
	}
	log.WithField("conferencecall", cc).Debugf("Found conferencecall info. conferencecall_id: %s", cc.ID)

	return h.updateStatus(ctx, cc.ID, status)
}

// UpdateStatusJoined is handy function for update the conferencecall's status to the joined.
// it increases corresponded counter
func (h *conferencecallHandler) UpdateStatusJoined(ctx context.Context, conferencecallID uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "UpdateStatusJoined",
			"conferencecall_id": conferencecallID,
		},
	)

	res, err := h.updateStatus(ctx, conferencecallID, conferencecall.StatusJoined)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateStatusLeaving is handy function for update the conferencecall's status to the leaved.
// it increases corresponded counter
func (h *conferencecallHandler) UpdateStatusLeaving(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "UpdateStatusLeaving",
			"conferencecall_id": id,
		},
	)

	res, err := h.updateStatus(ctx, id, conferencecall.StatusLeaving)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateStatusLeaved is handy function for update the conferencecall's status to the leaved.
// it increases corresponded counter
func (h *conferencecallHandler) UpdateStatusLeaved(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "UpdateStatusLeaved",
			"conferencecall_id": id,
		},
	)

	res, err := h.updateStatus(ctx, id, conferencecall.StatusLeaved)
	if err != nil {
		log.Errorf("Could not update the status correctly. err: %v", err)
		return nil, err
	}

	return res, nil

}
