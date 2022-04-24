package campaignhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
)

// UpdateStatusRun verifies the given campaign for run.
// if every condition is ok, it sets the status to run and starts the campaign execution.
func (h *campaignHandler) UpdateStatusRun(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "UpdateStatusRun",
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

	// check the campaign is valid
	if c.OutdialID == uuid.Nil {
		log.Infof("The campaign has no outdial_id.")
		return nil, fmt.Errorf("no outdial_id set")
	} else if c.OutplanID == uuid.Nil {
		log.Infof("The campaign has no outplan_id.")
		return nil, fmt.Errorf("no outplan_id set")
	}

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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignStatusRun, res)

	// execute campaign handle with 1 second delay
	if c.Execute != campaign.ExecuteRun {
		log.Debugf("Starting campaign execute.")
		if errExecute := h.reqHandler.CAV1CampaignExecute(ctx, id, 1000); errExecute != nil {
			log.Errorf("Could not execute the campaign correctly. Stopping the campaign. campaign_id: %s", id)
			_, _ = h.updateStatusStop(ctx, id)
			return nil, errExecute
		}
	}

	return res, nil
}

// // Run validates the campaign is run-able and make the campaign run if it is ready.
// func (h *campaignHandler) Run(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
// 	log := logrus.WithFields(
// 		logrus.Fields{
// 			"func": "Run",
// 			"id":   id,
// 		})
// 	log.Debug("Running the campaign.")

// 	// get campaign
// 	c, err := h.Get(ctx, id)
// 	if err != nil {
// 		log.Errorf("Could not get campaign. err: %v", err)
// 		return nil, err
// 	}

// 	if c.Status == campaign.StatusRun {
// 		log.Infof("Already status run. campaign_id: %s", c.ID)
// 		return c, nil
// 	}

// 	// check the campaign is valid
// 	if c.OutdialID == uuid.Nil {
// 		log.Infof("The campaign has no outdial_id.")
// 		return nil, fmt.Errorf("no outdial_id set")
// 	} else if c.OutplanID == uuid.Nil {
// 		log.Infof("The campaign has no outplan_id.")
// 		return nil, fmt.Errorf("no outplan_id set")
// 	}

// 	// Set status run
// 	res, errStatus := h.UpdateStatus(ctx, id, campaign.StatusRun)
// 	if errStatus != nil {
// 		log.Errorf("Could not update the campaign status running. err: %v", errStatus)
// 		return nil, errStatus
// 	}

// 	// start campaign handle
// 	// send running handle request

// 	return res, nil
// }

// isRunable returns true if a given campaign is run-able
func (h *campaignHandler) isRunable(ctx context.Context, c *campaign.Campaign) bool {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "isDialable",
			"campaign_id": c.ID,
		})
	log.Debug("Checking the campaign is run-able.")

	if c.OutdialID == uuid.Nil {
		log.Infof("The campaign has no outdial_id.")
		return false
	} else if c.OutplanID == uuid.Nil {
		log.Infof("The campaign has no outplan_id.")
		return false
	}

	return true
}

// getTargetDestination returns target destination
func (h *campaignHandler) getTargetDestination(ctx context.Context, target *omoutdialtarget.OutdialTarget, plan *outplan.Outplan) (*cmaddress.Address, int, int) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":           "getTargetDestination",
			"outdial_target": target,
		})
	log.Debug("Getting destination address.")

	maxTryCounts := []int{
		plan.MaxTryCount0,
		plan.MaxTryCount1,
		plan.MaxTryCount2,
		plan.MaxTryCount3,
		plan.MaxTryCount4,
	}

	tryCounts := []int{
		target.TryCount0,
		target.TryCount1,
		target.TryCount2,
		target.TryCount3,
		target.TryCount4,
	}

	destinations := []*cmaddress.Address{
		target.Destination0,
		target.Destination1,
		target.Destination2,
		target.Destination3,
		target.Destination4,
	}

	for i, maxTryCount := range maxTryCounts {
		if destinations[i] == nil {
			continue
		}

		if tryCounts[i] >= maxTryCount {
			continue
		}

		return destinations[i], i, tryCounts[i] + 1
	}

	// should not reach to here.
	log.Errorf("Something went wrong. Could not find dial destination.")
	return nil, 0, 0
}
