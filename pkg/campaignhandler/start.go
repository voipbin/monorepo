package campaignhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// StatusRunning running the
func (h *campaignHandler) StatusRunning(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Start",
			"id":   id,
		})
	log.Debug("Running campaign.")

	// get campaign
	tmpCampaign, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	// validate
	if tmpCampaign.OutdialID == uuid.Nil {
		return nil, fmt.Errorf("no outdial_id set")
	}
	if tmpCampaign.OutplanID == uuid.Nil {
		return nil, fmt.Errorf("no outplan_id set")
	}

	res, errStatus := h.UpdateStatus(ctx, id, campaign.StatusRunning)
	if errStatus != nil {
		log.Errorf("Could not update the campaign status running. err: %v", errStatus)
		return nil, errStatus
	}

	// start campaign handle
	// send running handle request

	return res, nil
}

// UpdateBasicInfo updates outplan's basic info
func (h *campaignHandler) Running(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Start",
			"id":   id,
		})
	log.Debug("Run campaign.")

	// // get campaign
	// c, err := h.Get(ctx, id)
	// if err != nil {
	// 	log.Errorf("Could not get campaign. err: %v", err)
	// 	return nil, err
	// }

	// // get outplan
	// p, err := h.outplanHandler.Get(ctx, c.OutplanID)
	// if err != nil {
	// 	log.Errorf("Could not get outplan. err: %v", err)
	// 	return nil, err
	// }

	// if c.QueueID == uuid.Nil {
	// 	// dial
	// }

	// // check
	// // get queue's available agents
	// agents, err := h.reqHandler.QMV1QueueGetAgents(ctx, c.QueueID, amagent.StatusAvailable)
	// if err != nil {
	// 	log.Errorf("Could not get available agents. err: %v", err)
	// 	return nil, err
	// }

	// // res, errStatus := h.UpdateStatus(ctx, id, campaign.StatusRunning)
	// // if errStatus != nil {
	// // 	log.Errorf("Could not update the campaign status running. err: %v", errStatus)
	// // 	return nil, errStatus
	// // }

	// // start campaign handle
	// // send start handle request

	// return res, nil
	return nil, nil
}

// UpdateBasicInfo updates outplan's basic info
func (h *campaignHandler) runningQueue(ctx context.Context, c *campaign.Campaign) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "runningQueue",
			"id":   c.ID,
		})
	log.Debug("Run campaign.")

	// // get plan
	// p, err := h.outplanHandler.Get(ctx, c.OutplanID)
	// if err != nil {
	// 	log.Errorf("Could not get outplan. err: %v", err)
	// 	return nil, err
	// }

	// check is dialable
	dialable, err := h.isDialable(ctx, c.ID, c.QueueID, c.ServiceLevel)
	if err != nil {
		log.Errorf("Could not verify the campaign is dialable. err: %v", err)
		return nil, err
	}

	if !dialable {
		// it's not dialable
		return nil, nil
	}

	return nil, nil

	// // create a queuecall
	// h.campaigncallHandler.Create(ctx, c.CustomerID, c.ID, )

	// agents, err := h.reqHandler.QMV1QueueGetAgents(ctx, c.QueueID, amagent.StatusAvailable)
	// if err != nil {
	// 	log.Errorf("Could not get available agents. err: %v", err)
	// 	return nil, err
	// }

	// // get queuecalls
	// qcs, err := h.campaigncallHandler.GetsByCampaignIDAndStatus(ctx, c.ID, campaigncall.StatusProgressing, dbhandler.DefaultTimeStamp, 100)
	// if err != nil {
	// 	log.Errorf("Could not get campaigncalls. err: %v", err)
	// 	return nil, err
	// }

	// // if len(qcs)

}

// isDialable returns true if a given campaign is dialable
func (h *campaignHandler) isDialable(ctx context.Context, campaignID, queueID uuid.UUID, serviceLevel int) (bool, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "isDialable",
			"id":   campaignID,
		})
	log.Debug("Run campaign.")

	if queueID == uuid.Nil {
		// the campaign has no queue_id.
		return true, nil
	}

	agents, err := h.reqHandler.QMV1QueueGetAgents(ctx, queueID, amagent.StatusAvailable)
	if err != nil {
		log.Errorf("Could not get available agents. err: %v", err)
		return false, err
	}

	// get queuecall dialings
	dialings, err := h.campaigncallHandler.GetsByCampaignIDAndStatus(ctx, campaignID, campaigncall.StatusDialing, dbhandler.DefaultTimeStamp, 100)
	if err != nil {
		log.Errorf("Could not get campaigncalls. err: %v", err)
		return false, err
	}

	dialingLen := len(dialings)
	agentCapacity := (len(agents) * serviceLevel) / 100.0
	if int(agentCapacity) <= dialingLen {
		return false, nil
	}

	return true, nil
}
