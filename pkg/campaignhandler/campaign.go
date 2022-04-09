package campaignhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// Create creates a new campaign
func (h *campaignHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*campaign.Campaign, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"customer_id": customerID,
		})

	ts := dbhandler.GetCurTime()
	id := uuid.Must(uuid.NewV4())
	t := &campaign.Campaign{
		ID:             id,
		CustomerID:     customerID,
		Name:           name,
		Detail:         detail,
		Status:         campaign.StatusStop,
		OutplanID:      outplanID,
		OutdialID:      outdialID,
		QueueID:        queueID,
		NextCampaignID: nextCampaignID,
		TMCreate:       ts,
		TMUpdate:       ts,
		TMDelete:       dbhandler.DefaultTimeStamp,
	}
	log.WithField("campaign", t).Debug("Creating a new campaign.")

	if err := h.db.CampaignCreate(ctx, t); err != nil {
		log.Errorf("Could not create the campaign. err: %v", err)
		return nil, err
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created campaign. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignCreated, res)

	return res, nil
}

// Delete delets the campaign
func (h *campaignHandler) Delete(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Delete",
			"campaign_id": id,
		})

	if errDelete := h.db.CampaignDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete campaign. err: %v", errDelete)
		return nil, errDelete
	}

	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted campaign. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignDeleted, res)

	return res, nil
}

// Get returns campaign
func (h *campaignHandler) Get(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Get",
			"campaign_id": id,
		})
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	return res, nil
}

// GetsByCustomerID returns list of campaigns
func (h *campaignHandler) GetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "GetsByCustomerID",
			"customer_id": customerID,
			"token":       token,
			"limit":       limit,
		})
	log.Debug("Getting campaigns.")

	res, err := h.db.CampaignGetsByCustomerID(ctx, customerID, token, limit)
	if err != nil {
		log.Errorf("Could not get campaigns. err: %v", err)
		return nil, err
	}

	return res, nil
}

// UpdateBasicInfo updates campaign's basic info
func (h *campaignHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "UpdateBasicInfo",
			"id":     id,
			"name":   name,
			"detail": detail,
		})
	log.Debug("Updating campaign basic info.")

	if err := h.db.CampaignUpdateBasicInfo(ctx, id, name, detail); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	return res, nil
}

// UpdateResourceInfo updates campaign's resource info
func (h *campaignHandler) UpdateResourceInfo(ctx context.Context, id, outplanID, outdialID, queueID uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":       "UpdateResourceInfo",
			"id":         id,
			"outplan_id": outplanID,
			"outdial_id": outdialID,
			"queue_id":   queueID,
		})
	log.Debug("Updating campaign basic info.")

	if err := h.db.CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, queueID); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	return res, nil
}

// UpdateNextCampaignID updates campaign's next_campaign_id info
func (h *campaignHandler) UpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":             "UpdateResourceInfo",
			"id":               id,
			"next_campaign_id": nextCampaignID,
		})
	log.Debug("Updating campaign next_campaign_id info.")

	if err := h.db.CampaignUpdateNextCampaignID(ctx, id, nextCampaignID); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	return res, nil
}

// UpdateStatus updates campaign's status
func (h *campaignHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":   "UpdateStatus",
			"id":     id,
			"status": status,
		})
	log.Debug("Updating campaign status.")

	if err := h.db.CampaignUpdateStatus(ctx, id, status); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	return res, nil
}

// UpdateServiceLevel updates campaign's service_level
func (h *campaignHandler) UpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":          "UpdateServiceLevel",
			"id":            id,
			"service_level": serviceLevel,
		})
	log.Debug("Updating campaign service_level.")

	if err := h.db.CampaignUpdateServiceLevel(ctx, id, serviceLevel); err != nil {
		log.Errorf("Could not update campaign service_level. err: %v", err)
		return nil, err
	}

	// get updated info
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}

	return res, nil
}
