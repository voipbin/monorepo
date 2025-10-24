package aicallhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-common-handler/models/identity"
)

// Create is creating a new aicall.
// it increases corresponded counter
func (h *aicallHandler) Create(
	ctx context.Context,
	c *ai.AI,
	activeflowID uuid.UUID,
	referenceType aicall.ReferenceType,
	referenceID uuid.UUID,
	confbridgeID uuid.UUID,
	gender aicall.Gender,
	language string,
) (*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Create",
		"ai":   c,
	})

	id := h.utilHandler.UUIDCreate()
	tmp := &aicall.AIcall{
		Identity: identity.Identity{
			ID:         id,
			CustomerID: c.CustomerID,
		},

		AIID:          c.ID,
		AIEngineType:  c.EngineType,
		AIEngineModel: c.EngineModel,
		AIEngineData:  c.EngineData,

		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		ConfbridgeID: confbridgeID,

		Gender:   gender,
		Language: language,

		Status: aicall.StatusInitiating,
	}
	log = log.WithField("aicall_id", id.String())
	log.WithField("aicall", tmp).Debugf("Creating aicall. aicall_id: %s", tmp.ID)

	if errCreate := h.db.AIcallCreate(ctx, tmp); errCreate != nil {
		log.Errorf("Could not create a new aicall. err: %v", errCreate)
		return nil, errCreate
	}
	promAIcallCreateTotal.WithLabelValues(string(tmp.ReferenceType)).Inc()

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get created aicall info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusInitializing, res)

	// todo: start health check

	return res, nil
}

// Get is handy function for getting a aicall.
func (h *aicallHandler) Get(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Get",
			"aicall_id": id,
		},
	)

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get aicall info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// Delete deletes the aicall.
func (h *aicallHandler) Delete(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":      "Delete",
			"aicall_id": id,
		},
	)

	if err := h.db.AIcallDelete(ctx, id); err != nil {
		log.Errorf("Could not delete the aicall. err: %v", err)
		return nil, err
	}

	res, err := h.db.AIcallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not updated aicall. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID returns a aicall by the reference_id.
func (h *aicallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.db.AIcallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// // UpdateStatusPausing updates the status to pausing
// func (h *aicallHandler) UpdateStatusPausing(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {

// 	if errUpdate := h.db.AIcallUpdateStatusPausing(ctx, id); errUpdate != nil {
// 		return nil, errors.Wrapf(errUpdate, "could not update the status to pausing. aicall_id: %s", id)
// 	}

// 	res, err := h.Get(ctx, id)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
// 	}
// 	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusPausing, res)

// 	return res, nil
// }

// // UpdateStatusResuming updates the status to resuming
// func (h *aicallHandler) UpdateStatusResuming(ctx context.Context, id uuid.UUID, confbridgeID uuid.UUID) (*aicall.AIcall, error) {

// 	if errUpdate := h.db.AIcallUpdateStatusResuming(ctx, id, confbridgeID); errUpdate != nil {
// 		return nil, errors.Wrapf(errUpdate, "could not update the status to resuming. aicall_id: %s", id)
// 	}

// 	res, err := h.Get(ctx, id)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
// 	}
// 	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusResuming, res)

// 	return res, nil
// }

// // UpdateStatusTerminating updates the status to terminating
// func (h *aicallHandler) UpdateStatusTerminating(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {

// 	if errUpdate := h.db.AIcallUpdateStatusTerminating(ctx, id); errUpdate != nil {
// 		return nil, errors.Wrapf(errUpdate, "could not update the status to terminating. aicall_id: %s", id)
// 	}

// 	res, err := h.Get(ctx, id)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
// 	}
// 	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusTerminating, res)

// 	return res, nil
// }

// // UpdateStatusTerminated updates the status to end
// func (h *aicallHandler) UpdateStatusTerminated(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {

// 	if errUpdate := h.db.AIcallUpdateStatusTerminated(ctx, id); errUpdate != nil {
// 		return nil, errors.Wrapf(errUpdate, "could not update the status to terminated. aicall_id: %s", id)
// 	}

// 	res, err := h.Get(ctx, id)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
// 	}
// 	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusTerminated, res)

// 	return res, nil
// }

// Gets returns list of aicalls.
func (h *aicallHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*aicall.AIcall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "Gets",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.AIcallGets(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get aicalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

func (h *aicallHandler) UpdatePipecatcallID(ctx context.Context, id uuid.UUID, pipecatcallID uuid.UUID) error {
	return h.db.AIcallUpdatePipecatcallID(ctx, id, pipecatcallID)
}

// UpdateStatusTerminating updates the status to terminating
func (h *aicallHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status aicall.Status) (*aicall.AIcall, error) {

	if errUpdate := h.db.AIcallUpdateStatus(ctx, id, status); errUpdate != nil {
		return nil, errors.Wrapf(errUpdate, "could not update the status to terminating. aicall_id: %s", id)
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get updated aicall info. aicall_id: %s", id)
	}

	switch res.Status {
	case aicall.StatusProgressing:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusProgressing, res)
	case aicall.StatusPausing:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusPausing, res)
	case aicall.StatusResuming:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusResuming, res)
	case aicall.StatusTerminating:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusTerminating, res)
	case aicall.StatusTerminated:
		h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, aicall.EventTypeStatusTerminated, res)
	}

	return res, nil
}
