package campaignhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaign"
)

// campaignRun verifies the given campaign for run.
// if every condition is ok, it sets the status to run and starts the campaign execution.
func (h *campaignHandler) campaignRun(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "campaignRun",
		"id":   id,
	})
	log.Debug("Updating the campaign status to run.")

	// get campaign
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	if c.Status == campaign.StatusRun {
		log.Infof("Already status run. campaign_id: %s", c.ID)
		return c, nil
	}

	// check the campaign resource is valid
	if !h.validateResources(ctx, c.ID, c.CustomerID, c.OutplanID, c.OutdialID, c.QueueID, c.NextCampaignID) {
		log.Errorf("The campaign resource is not valid.")
		return nil, fmt.Errorf("the campaign resource is not valid")
	}

	// // check the campaign is runable
	// if !h.isRunable(ctx, c) {
	// 	log.Errorf("The campaign is not runnable.")
	// 	return nil, fmt.Errorf("campaign is not runnable")
	// }

	// Set status run
	if err := h.db.CampaignUpdateStatusAndExecute(ctx, id, campaign.StatusRun, campaign.ExecuteRun); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	promCampaignStatusRunTotal.Inc()
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignStatusRun, res)

	// execute campaign handle with 1 second delay
	if c.Execute != campaign.ExecuteRun {
		log.Debugf("Starting campaign execute.")
		if errExecute := h.reqHandler.CampaignV1CampaignExecute(ctx, id, 1000); errExecute != nil {
			log.Errorf("Could not execute the campaign correctly. Stopping the campaign. campaign_id: %s", id)
			_, _ = h.campaignStopNow(ctx, id)
			return nil, errExecute
		}
	}

	return res, nil
}
