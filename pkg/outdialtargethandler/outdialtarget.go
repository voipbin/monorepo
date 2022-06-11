package outdialtargethandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
	"gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/dbhandler"
)

// Create creates a new outdial
func (h *outdialTargetHandler) Create(
	ctx context.Context,
	outdialID uuid.UUID,
	name string,
	detail string,
	data string,
	destination0 *commonaddress.Address,
	destination1 *commonaddress.Address,
	destination2 *commonaddress.Address,
	destination3 *commonaddress.Address,
	destination4 *commonaddress.Address,
) (*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Create",
			"outdial_id": outdialID,
		})

	id := uuid.Must(uuid.NewV4())
	ts := dbhandler.GetCurTime()
	t := &outdialtarget.OutdialTarget{
		ID:        id,
		OutdialID: outdialID,

		Name:   name,
		Detail: detail,

		Data:   data,
		Status: outdialtarget.StatusIdle,

		Destination0: destination0,
		Destination1: destination1,
		Destination2: destination2,
		Destination3: destination3,
		Destination4: destination4,

		TryCount0: 0,
		TryCount1: 0,
		TryCount2: 0,
		TryCount3: 0,
		TryCount4: 0,

		TMCreate: ts,
		TMUpdate: ts,
		TMDelete: dbhandler.DefaultTimeStamp,
	}
	log.WithField("outdial_target", t).Debug("Creating a new outdialtarget.")

	if err := h.db.OutdialTargetCreate(ctx, t); err != nil {
		log.Errorf("Could not create the outdial. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created outdial. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns outdialtarget
func (h *outdialTargetHandler) Get(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "Get",
			"outdial_target_id": id,
		})

	res, err := h.db.OutdialTargetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not create the outdial. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes outdialtarget
func (h *outdialTargetHandler) Delete(ctx context.Context, id uuid.UUID) (*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":              "Delete",
			"outdial_target_id": id,
		})

	if err := h.db.OutdialTargetDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the outdial. err: %v", err)
		return nil, err
	}

	res, err := h.db.OutdialTargetGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted the outdial. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByOutdialID returns list of outdialtargets
func (h *outdialTargetHandler) GetsByOutdialID(ctx context.Context, outdialID uuid.UUID, token string, limit uint64) ([]*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "GetsByOutdialID",
			"outdial_id": outdialID,
		})

	res, err := h.db.OutdialTargetGetsByOutdialID(ctx, outdialID, token, limit)
	if err != nil {
		log.Errorf("Could not create the outdial. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetAvailable returns outdialtarget
func (h *outdialTargetHandler) GetAvailable(
	ctx context.Context,
	outdialID uuid.UUID,
	tryCount0 int,
	tryCount1 int,
	tryCount2 int,
	tryCount3 int,
	tryCount4 int,
	limit uint64,
) ([]*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Get",
			"outdial_id": outdialID,
		})

	res, err := h.db.OutdialTargetGetAvailable(
		ctx,
		outdialID,
		tryCount0,
		tryCount1,
		tryCount2,
		tryCount3,
		tryCount4,
		limit,
	)
	if err != nil {
		log.Errorf("Could not get available outdialtarget. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateProgressing updates the outdialtarget's status to progress and increase try count
func (h *outdialTargetHandler) UpdateProgressing(ctx context.Context, id uuid.UUID, destinationIndex int) (*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":             "UpdateProgressing",
			"outdialtarget_id": id,
		})

	if errUpdate := h.db.OutdialTargetUpdateProgressing(ctx, id, destinationIndex); errUpdate != nil {
		log.Errorf("Could not update the outdial target status to progressing. err: %v", errUpdate)
		return nil, errUpdate
	}

	// get updated
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outdialtarget. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateProgressing updates the outdialtarget's status to progress and increase try count
func (h *outdialTargetHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status outdialtarget.Status) (*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":             "UpdateStatus",
			"outdialtarget_id": id,
			"status":           status,
		})

	if errUpdate := h.db.OutdialTargetUpdateStatus(ctx, id, status); errUpdate != nil {
		log.Errorf("Could not update the outdialtarget status. err: %v", errUpdate)
		return nil, errUpdate
	}

	// get updated
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outdialtarget. err: %v", err)
		return nil, err
	}

	return res, nil
}
