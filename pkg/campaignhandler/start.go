package campaignhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
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

func (h *campaignHandler) Running(ctx context.Context, id uuid.UUID) error {
	if err := h.running(ctx, id); err != nil {
		// send request again after 5 sec

		return nil
	}

	// send request right now.

	return nil
}

// running
func (h *campaignHandler) running(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "running",
			"campaign_id": id,
		})
	log.Debug("Run campaign.")

	// get campaign
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return err
	}

	// check is dialable
	if !h.isDialable(ctx, c.ID, c.QueueID, c.ServiceLevel) {
		log.Debugf("Campaign is not dialble.")
		return fmt.Errorf("campaign is not dialable")
	}

	// get outplan
	p, err := h.outplanHandler.Get(ctx, c.OutplanID)
	if err != nil {
		log.Errorf("Could not get outplan. err: %v", err)
		return err
	}

	// get outdial target
	targets, err := h.reqHandler.OMV1OutdialtargetGetsAvailable(
		ctx,
		c.OutdialID,
		p.MaxTryCount0,
		p.MaxTryCount1,
		p.MaxTryCount2,
		p.MaxTryCount3,
		p.MaxTryCount4,
		p.TryInterval,
		1,
	)
	if err != nil {
		log.Errorf("Could not get outdial target. err: %v", err)
		return err
	}

	if len(targets) == 0 {
		// could not find available outdial target.
		return fmt.Errorf("could not find available outdial target")
	}

	// get dial destination
	destination, destinationIndex, tryCount := h.getTargetDestination(ctx, &targets[0], p)
	if destination == nil {
		log.WithField("target", targets[0]).Errorf("Could not find target destination.")
		return fmt.Errorf("could not find target destination")
	}

	callID := uuid.Must(uuid.NewV4())

	// create campaigncall
	campaignCall, err := h.campaigncallHandler.Create(
		ctx,
		c.CustomerID,
		c.ID,
		c.OutplanID,
		c.OutdialID,
		targets[0].ID,
		c.QueueID,
		uuid.Nil,
		campaigncall.ReferenceTypeCall,
		callID,
		p.Source,
		destination,
		destinationIndex,
		tryCount,
	)
	if err != nil {
		log.Errorf("Could not create a campaign call. err: %v", err)
		return err
	}

	// create a call
	newCall, err := h.reqHandler.CMV1CallCreateWithID(ctx, callID, c.CustomerID, p.FlowID, uuid.Nil, p.Source, destination)
	if err != nil {
		// update camapaign call to fail
		h.campaigncallHandler.UpdateStatus(ctx, campaignCall.ID, campaigncall.StatusProgressing)
		log.Errorf("Could not create a call. err: %v", err)
		return err
	}

	// callID := uuid.Must(uuid.NewV4())
	// campaignCallID := uuid.Must(uuid.NewV4())

	// h.reqHandler.CMV1CallCreateWithID(ctx, cam)

	// // get destiantion
	// destination := h.getDestination(ctx, &targets[0])
	// calls, err := h.reqHandler.CMV1CallsCreate(
	// 	ctx,
	// 	c.CustomerID,
	// 	p.FlowID,
	// 	uuid.Nil,
	// 	p.Source,
	// 	[]cmaddress.Address{*destination},
	// )

	// if err != nil {
	// 	log.Errorf("Could not create a outgoing call. err: %v", err)
	// }

	// // check
	// // get queue's available agents
	// agents, err := h.reqHandler.QMV1QueueGetAgents(ctx, c.QueueID, amagent.StatusAvailable)
	// if err != nil {
	// 	log.Errorf("Could not get available agents. err: %v", err)
	// 	return err
	// }

	// // res, errStatus := h.UpdateStatus(ctx, id, campaign.StatusRunning)
	// // if errStatus != nil {
	// // 	log.Errorf("Could not update the campaign status running. err: %v", errStatus)
	// // 	return nil, errStatus
	// // }

	// // start campaign handle
	// // send start handle request

	// return res, nil
	return nil
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
func (h *campaignHandler) isDialable(ctx context.Context, campaignID, queueID uuid.UUID, serviceLevel int) bool {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "isDialable",
			"id":   campaignID,
		})
	log.Debug("Run campaign.")

	if queueID == uuid.Nil {
		// the campaign has no queue_id.
		return true
	}

	agents, err := h.reqHandler.QMV1QueueGetAgents(ctx, queueID, amagent.StatusAvailable)
	if err != nil {
		log.Errorf("Could not get available agents. err: %v", err)
		return false
	}

	// get queuecall dialings
	dialings, err := h.campaigncallHandler.GetsByCampaignIDAndStatus(ctx, campaignID, campaigncall.StatusDialing, dbhandler.DefaultTimeStamp, 100)
	if err != nil {
		log.Errorf("Could not get campaigncalls. err: %v", err)
		return false
	}

	dialingLen := len(dialings)
	agentCapacity := (len(agents) * serviceLevel) / 100.0

	if int(agentCapacity) <= dialingLen {
		// currerntly campaign
		return false
	}
	log.Debugf("The campaign is dialable. agent_capacity: %d, dialing_len: %d", int(agentCapacity), dialingLen)

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
