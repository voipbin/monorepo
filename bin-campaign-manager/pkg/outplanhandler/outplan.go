package outplanhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
)

// Create creates a new outplan
func (h *outplanHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,

	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,

	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*outplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	id := h.util.UUIDCreate()
	t := &outplan.Outplan{
		ID:         id,
		CustomerID: customerID,

		Name:   name,
		Detail: detail,

		Source: source,

		DialTimeout: dialTimeout,
		TryInterval: tryInterval,

		MaxTryCount0: maxTryCount0,
		MaxTryCount1: maxTryCount1,
		MaxTryCount2: maxTryCount2,
		MaxTryCount3: maxTryCount3,
		MaxTryCount4: maxTryCount4,
	}
	log.WithField("outplan", t).Debug("Creating a new outplan.")

	if err := h.db.OutplanCreate(ctx, t); err != nil {
		log.Errorf("Could not create the outplan. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created outplan. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Get returns outplan
func (h *outplanHandler) Get(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"outplan_id": id,
	})

	res, err := h.db.OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete delets the outplan
func (h *outplanHandler) Delete(ctx context.Context, id uuid.UUID) (*outplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Delete",
		"outplan_id": id,
	})

	if errDelete := h.db.OutplanDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete outplan. err: %v", errDelete)
		return nil, errDelete
	}

	res, err := h.db.OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted outplan. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByCustomerID returns list of outplans
func (h *outplanHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByCustomerID",
		"customer_id": customerID,
		"token":       token,
		"limit":       limit,
	})
	log.Debug("Getting outplans.")

	res, err := h.db.OutplanGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get outplans. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateBasicInfo updates outplan's basic info
func (h *outplanHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*outplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "UpdateBasicInfo",
		"id":     id,
		"name":   name,
		"detail": detail,
	})
	log.Debug("Updating outplan basic info.")

	if err := h.db.OutplanUpdateBasicInfo(ctx, id, name, detail); err != nil {
		log.Errorf("Could not update outplans. err: %v", err)
		return nil, err
	}

	// get updated outplan
	res, err := h.db.OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outplan info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateActionInfo updates outplan's dial info
func (h *outplanHandler) UpdateDialInfo(
	ctx context.Context,
	id uuid.UUID,
	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*outplan.Outplan, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "UpdateDialInfo",
		"id":              id,
		"source":          source,
		"dial_timeout":    dialTimeout,
		"try_interval":    tryInterval,
		"mac_try_count_0": maxTryCount0,
		"mac_try_count_1": maxTryCount1,
		"mac_try_count_2": maxTryCount2,
		"mac_try_count_3": maxTryCount3,
		"mac_try_count_4": maxTryCount4,
	})
	log.Debug("Updating outplan dial info.")

	if err := h.db.OutplanUpdateDialInfo(
		ctx,
		id,
		source,
		dialTimeout,
		tryInterval,
		maxTryCount0,
		maxTryCount1,
		maxTryCount2,
		maxTryCount3,
		maxTryCount4,
	); err != nil {
		log.Errorf("Could not update outplan dial info. err: %v", err)
		return nil, err
	}

	// get updated outplan
	res, err := h.db.OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated outplan info. err: %v", err)
		return nil, err
	}

	return res, nil
}
