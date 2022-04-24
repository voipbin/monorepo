package campaignhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// UpdateStatusStopping
func (h *campaignHandler) UpdateStatusStopping(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "UpdateStatusStopping",
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

// updateStatusStop updates the campaign's status to stop.
func (h *campaignHandler) updateStatusStop(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "updateStatusStop",
			"id":   id,
		})
	log.Debug("Updating the campaign status to stop.")

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

	if c.Status != campaign.StatusStopping {
		log.Errorf("The campaign's status is not stopping.")
		return nil, fmt.Errorf("wrong status")
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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignStatusStop, res)

	return res, nil
}

func (h *campaignHandler) isStopable(ctx context.Context, id uuid.UUID) bool {
	log := logrus.WithFields(
		logrus.Fields{
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

	if c.Status != campaign.StatusStopping {
		// the only stopping status can go to the stop.
		return false
	}

	// get campaign calls
	ccs, err := h.campaigncallHandler.GetsOngoingByCampaignID(ctx, c.ID, dbhandler.GetCurTime(), 10)
	if err != nil {
		log.Errorf("Could not get ongoing campaigncalls. err: %s", err)
	}

	if len(ccs) > 0 {
		return false
	}

	return true
}
