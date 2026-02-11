package campaignhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaign"
)

// UpdateStatusStopping updates the campaign's status to the stopping.
// it checks the condition for status update and returns updated campaign and error
func (h *campaignHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status campaign.Status) (*campaign.Campaign, error) {
	switch status {
	case campaign.StatusRun:
		return h.campaignRun(ctx, id)
	case campaign.StatusStop:
		return h.campaignStop(ctx, id)

	default:
		return nil, fmt.Errorf("unsupported status. status: %s", status)
	}
}

// UpdateStatusStopping updates the campaign's status to the stopping.
// it checks the condition for status update and returns updated campaign and error
func (h *campaignHandler) campaignStop(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "campaignStop",
		"id":   id,
	})
	log.Debug("Updating the campaign status to stopping.")

	var res *campaign.Campaign
	var err error
	if h.isStoppable(ctx, id) {
		log.Debugf("The campaign is stoppable. Stop now. campaign_id: %s", id)
		res, err = h.campaignStopNow(ctx, id)
	} else {
		res, err = h.campaignStopping(ctx, id)
	}
	if err != nil {
		log.Errorf("Could not stop the campaign. err: %v", err)
		return nil, err
	}

	return res, nil
}

// campaignStopping updates the campaign's status to the stopping.
// it checks the condition for status update and returns updated campaign and error
func (h *campaignHandler) campaignStopping(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "campaignStopping",
		"id":   id,
	})
	log.Debug("Updating the campaign status to stopping.")

	// get campaign
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	if c.Status == campaign.StatusStop || c.Status == campaign.StatusStopping {
		log.Infof("Status is already stop or stopping. campaign_id: %s, status: %s", c.ID, c.Status)
		return c, nil
	}

	// Set status stopping
	if err := h.db.CampaignUpdateStatus(ctx, id, campaign.StatusStopping); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignStatusStopping, res)

	return res, nil
}

// campaignStopNow updates the campaign's status to stop.
func (h *campaignHandler) campaignStopNow(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "campaignStopNow",
		"id":   id,
	})
	log.Debug("Updating the campaign status to stop now.")

	// get campaign
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	if c.Status == campaign.StatusStop {
		// already stopped.
		return c, nil
	}

	// Set status stop
	if err := h.db.CampaignUpdateStatus(ctx, id, campaign.StatusStop); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	promCampaignStatusStopTotal.Inc()
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignStatusStop, res)

	return res, nil
}

// isStoppable returns true if the campaign is stoppable
func (h *campaignHandler) isStoppable(ctx context.Context, id uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "isStopable",
		"campaign_id": id,
	})

	// get campaign
	c, err := h.Get(ctx, id)
	if err != nil {
		// could not get campaign info.
		log.Errorf("Could not get campaign. err: %v", err)
		return false
	}

	if c.Execute != campaign.ExecuteStop {
		// campaign exeuction is still running.
		return false
	}

	// get campaign calls
	ccs, err := h.campaigncallHandler.ListOngoingByCampaignID(ctx, c.ID, h.util.TimeGetCurTime(), 1)
	if err != nil {
		log.Errorf("Could not get ongoing campaigncalls. err: %s", err)
	}

	if len(ccs) > 0 {
		return false
	}

	return true
}
