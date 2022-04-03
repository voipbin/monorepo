package outdialtargethandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

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
	destination0 *cmaddress.Address,
	destination1 *cmaddress.Address,
	destination2 *cmaddress.Address,
	destination3 *cmaddress.Address,
	destination4 *cmaddress.Address,
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
	interval time.Duration,
	limit uint64,
) ([]*outdialtarget.OutdialTarget, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "Get",
			"outdial_id": outdialID,
		})

	ts := dbhandler.GetCurTimeAdd(-interval)
	res, err := h.db.OutdialTargetGetAvailable(
		ctx,
		outdialID,
		tryCount0,
		tryCount1,
		tryCount2,
		tryCount3,
		tryCount4,
		ts,
		limit,
	)
	if err != nil {
		log.Errorf("Could not get available outdialtarget. err: %v", err)
		return nil, err
	}

	return res, nil
}
