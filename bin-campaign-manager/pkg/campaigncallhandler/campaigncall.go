package campaigncallhandler

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaigncall"
)

// Create creates a new campaigncall
func (h *campaigncallHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,

	campaignID uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	outdialTargetID uuid.UUID,
	queueID uuid.UUID,

	activeflowID uuid.UUID,
	flowID uuid.UUID,

	referenceType campaigncall.ReferenceType,
	referenceID uuid.UUID,
	source *commonaddress.Address,
	destination *commonaddress.Address,
	destinationIndex int,
	tryCount int,
) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	id := h.util.UUIDCreate()
	t := &campaigncall.Campaigncall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},

		CampaignID:      campaignID,
		OutplanID:       outplanID,
		OutdialID:       outdialID,
		OutdialTargetID: outdialTargetID,
		QueueID:         queueID,

		ActiveflowID: activeflowID,
		FlowID:       flowID,

		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		Status:           campaigncall.StatusDialing,
		Source:           source,
		Destination:      destination,
		DestinationIndex: destinationIndex,
		TryCount:         tryCount,
	}
	log.WithField("campaigncall", t).Debug("Creating a new campaigncall.")

	if err := h.db.CampaigncallCreate(ctx, t); err != nil {
		log.Errorf("Could not create the campaigncall. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created campaigncall. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaigncall.EventTypeCampaigncallCreated, res)

	// set the outdial target status to progressing
	tmpTarget, err := h.reqHandler.OutdialV1OutdialtargetUpdateStatusProgressing(ctx, t.OutdialTargetID, destinationIndex)
	if err != nil {
		log.Errorf("Could not update the outdialtarget status to progressing. err: %v", err)
		_, _ = h.Done(ctx, t.ID, campaigncall.ResultFail)
		return nil, err
	}
	log.WithField("target", tmpTarget).Infof("Updated outdial target status to progressing. outdialtarget_id: %s", tmpTarget.ID)

	return res, nil
}

// Get returns list of campaigncall
func (h *campaigncallHandler) Get(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "Get",
		"id":   id,
	})
	log.Debug("Getting campaigncall.")

	res, err := h.db.CampaigncallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaigncall. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetByReferenceID returns list of campaigncall of the referenceID
func (h *campaigncallHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*campaigncall.Campaigncall, error) {
	// we don't write log here. it makes lots of noises.
	res, err := h.db.CampaigncallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetByActiveflowID returns list of campaigncall of the activeflowID
func (h *campaigncallHandler) GetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*campaigncall.Campaigncall, error) {
	// we don't write log here. it makes lots of noises.
	res, err := h.db.CampaigncallGetByActiveflowID(ctx, activeflowID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// List returns list of campaigncalls with filters
func (h *campaigncallHandler) List(ctx context.Context, token string, limit uint64, filters map[campaigncall.Field]any) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"token":   token,
		"limit":   limit,
		"filters": filters,
	})
	log.Debug("Getting campaigncalls with filters.")

	res, err := h.db.CampaigncallList(ctx, token, limit, filters)
	if err != nil {
		log.Errorf("Could not get campaigncalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ListByCustomerID returns list of campaigncall
func (h *campaigncallHandler) ListByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByCustomerID",
		"campaign_id": customerID,
		"token":       token,
		"limit":       limit,
	})
	log.Debug("Getting campaigncalls.")

	res, err := h.db.CampaigncallListByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get campaigncalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ListByCampaignID returns list of campaigncall
func (h *campaigncallHandler) ListByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsByCampaignID",
		"campaign_id": campaignID,
		"token":       token,
		"limit":       limit,
	})
	log.Debug("Getting campaigncalls.")

	res, err := h.db.CampaigncallListByCampaignID(ctx, campaignID, token, limit)
	if err != nil {
		log.Errorf("Could not get campaigncalls. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ListByCampaignIDAndStatus returns list of campaigncalls
func (h *campaigncallHandler) ListByCampaignIDAndStatus(ctx context.Context, campaignID uuid.UUID, status campaigncall.Status, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCampaignIDAndStatus",
			"campaign_id": campaignID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting campaigncalls.")

	res, err := h.db.CampaigncallListByCampaignIDAndStatus(ctx, campaignID, status, token, limit)
	if err != nil {
		log.Errorf("Could not get GetsByCampaignIDAndStatus. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsOngoingByCampaignID returns list of ongoing campaigncalls
func (h *campaigncallHandler) ListOngoingByCampaignID(ctx context.Context, campaignID uuid.UUID, token string, limit uint64) ([]*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "GetsOngoingByCampaignID",
		"campaign_id": campaignID,
		"token":       token,
		"limit":       limit,
	})
	log.Debug("Getting campaigncalls.")

	res, err := h.db.CampaigncallListOngoingByCampaignID(ctx, campaignID, token, limit)
	if err != nil {
		log.Errorf("Could not get GetsOngoingByCampaignID. err: %v", err)
		return nil, err
	}

	return res, nil
}

// updateStatus updates the status
func (h *campaigncallHandler) updateStatus(ctx context.Context, id uuid.UUID, status campaigncall.Status) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "updateStatus",
		"id":   id,
	})
	log.Debug("Getting campaigncalls.")

	if err := h.db.CampaigncallUpdateStatus(ctx, id, status); err != nil {
		log.Errorf("Could not get UpdateStatus. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaigncall. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaigncall.EventTypeCampaigncallUpdated, res)

	return res, nil
}

// updateStatusDone updates the status
func (h *campaigncallHandler) updateStatusDone(ctx context.Context, id uuid.UUID, result campaigncall.Result) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "updateStatusDone",
			"id":   id,
		})
	log.Debug("Getting campaigncalls.")

	if err := h.db.CampaigncallUpdateStatusAndResult(ctx, id, campaigncall.StatusDone, result); err != nil {
		log.Errorf("Could not get UpdateStatus. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaigncall. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaigncall.EventTypeCampaigncallUpdated, res)

	return res, nil
}

// Delete deletes the campaigncall
func (h *campaigncallHandler) Delete(ctx context.Context, id uuid.UUID) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "Delete",
		"campaigncall_id": id,
	})
	log.Debugf("Deleting a campaigncall. campaigncall_id: %s", id)

	c, err := h.db.CampaigncallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	if c.Status != campaigncall.StatusDone {
		log.Errorf("The campaigncall is not stop. status: %s", c.Status)
		return nil, err
	}
	log.WithField("campaigncall", c).Debugf("Deleting campaigncall. campaigncall_id: %s", c.ID)

	if errDelete := h.db.CampaigncallDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete campaign. err: %v", errDelete)
		return nil, errDelete
	}

	res, err := h.db.CampaigncallGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted campaigncall. err: %v", err)
		return nil, err
	}
	log.WithField("campaigncall", res).Debugf("Deleted campaigncall. campaign_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaigncall.EventTypeCampaigncallDeleted, res)

	return res, nil
}
