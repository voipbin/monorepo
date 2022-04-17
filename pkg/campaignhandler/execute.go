package campaignhandler

import (
	"context"
	"fmt"

	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// Execute exeuctes the campaign.
func (h *campaignHandler) Execute(ctx context.Context, id uuid.UUID) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Execute",
			"campaign_id": id,
		},
	)

	// get campaign
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. Stopping the campaign. err: %v", err)
		// todo: send the campaign stopping request
		return
		// return fmt.Errorf("could not get campaign. err: %v", err)
	}

	// check the campaign's status
	if c.Status != campaign.StatusRun {
		log.WithField("campaign", c).Infof("The campaign status is not run. Stop to execute. campaign_id: %s, status: %s", c.ID, c.Status)
		return
	}

	// check is dial-able
	if !h.isDialable(ctx, c.ID, c.QueueID, c.ServiceLevel) {
		log.Debugf("Campaign is not dialable now.")
		// send an execute request with 5 seconds of delay
		return
	}

	// get outplan
	p, err := h.outplanHandler.Get(ctx, c.OutplanID)
	if err != nil {
		log.Errorf("Could not get outplan. err: %v", err)
		// send an campaign stop request.
		return
	}

	// get available outdial target
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
		log.Errorf("Could not get available outdial target. err: %v", err)

		// send an campaign stop request
		return
	}

	if len(targets) == 0 {
		// no more available outdial target left
		if c.EndHandle == campaign.EndHandleStop {
			log.Info("The campaign has no outdial target left and end handle is stop.")
			// send an campaign stop request
			return
		}

		// send a campaign execute request with 5 seconds of delay
		return
	}

	dialed := false
	if c.QueueID != uuid.Nil {
		dialed, err = h.executeQueue(ctx, c, p, &targets[0])
	} else {
		dialed, err = h.executeNormal(ctx, c, p, &targets[0])
	}
	if err != nil {
		log.Infof("Could not execute the campaign correctly. Stopping campaign. err: %v", err)
		// send the campaign stopping request
		return
	}

	// if !dialed {
	// 	// send run request with 5 seconds
	// } else {
	// 	// send request right now.
	// }

	return
}

// executeQueue handles campaign's exeucte with a queue.
// it checks campaign's available outdial target and make a call.
func (h *campaignHandler) executeQueue(ctx context.Context, c *campaign.Campaign, p *outplan.Outplan, target *outdialtarget.OutdialTarget) (bool, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "executeQueue",
			"campaign_id": c.ID,
		})
	log.Debug("Execute campaign.")

	destination, destinationIndex, tryCount := h.getTargetDestination(ctx, target, p)
	if destination == nil {
		log.WithField("target", target).Error("Something was wrong. Could not find target destination.")
		return false, fmt.Errorf("could not find target destination")
	}

	// create call_id
	callID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	// create campaigncall
	campaignCall, err := h.campaigncallHandler.Create(
		ctx,
		c.CustomerID,
		c.ID,
		c.OutplanID,
		c.OutdialID,
		target.ID,
		c.QueueID,

		activeflowID,
		c.FlowID,

		campaigncall.ReferenceTypeCall,
		callID,
		p.Source,
		destination,
		destinationIndex,
		tryCount,
	)
	if err != nil {
		log.Errorf("Could not create a campaign call. err: %v", err)
		return false, err
	}

	// set the outdial target status to progressing
	tmpTarget, err := h.reqHandler.OMV1OutdialtargetUpdateStatusProgressing(ctx, target.ID, destinationIndex)
	if err != nil {
		log.Errorf("Could not update the outdialtarget status to progress. err: %v", err)
		return false, err
	}
	log.WithField("target", tmpTarget).Infof("Updated outdial target status to progressing. outdialtarget_id: %s", tmpTarget.ID)

	// create a call
	newCall, err := h.reqHandler.CMV1CallCreateWithID(ctx, callID, c.CustomerID, c.FlowID, activeflowID, uuid.Nil, p.Source, destination)
	if err != nil {
		// update camapaign call to fail
		h.campaigncallHandler.Done(ctx, campaignCall.ID, campaigncall.StatusDone, campaigncall.ResultFail)
		log.Errorf("Could not create a call. err: %v", err)
		return false, err
	}

	// update campaigncall's activeflow_id
	tmpCampaignCall, err := h.campaigncallHandler.UpdateActiveflowID(ctx, campaignCall.ID, newCall.ActiveFlowID)
	if err != nil {
		log.Errorf("Could not update the campaigncall activeflow id. err: %v", err)
		// we failed to update the activeflow_id, but just keep going.
	}
	log.WithField("campaigncall", tmpCampaignCall).Debugf("Updated activeflow id. activeflow_id: %s", newCall.ActiveFlowID)

	return true, nil
}

// run handles campaign's run status.
// it checks campaign's available outdial target and make a call.
func (h *campaignHandler) executeNormal(ctx context.Context, c *campaign.Campaign, p *outplan.Outplan, target *outdialtarget.OutdialTarget) (bool, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "executeNormal",
			"campaign_id": c.ID,
		})
	log.Debug("Run campaign.")

	// get dial destination
	destination, destinationIndex, tryCount := h.getTargetDestination(ctx, target, p)
	if destination == nil {
		log.WithField("target", target).Error("Something was wrong. Could not find target destination.")
		return false, fmt.Errorf("could not find target destination")
	}

	// create activeflow_id
	activeflowID := uuid.Must(uuid.NewV4())

	// create a campaigncall
	campaignCall, err := h.campaigncallHandler.Create(
		ctx,
		c.CustomerID,
		c.ID,
		c.OutplanID,
		c.OutdialID,
		target.ID,
		c.QueueID,

		activeflowID,
		c.FlowID,

		campaigncall.ReferenceTypeCall,
		uuid.Nil,
		p.Source,
		destination,
		destinationIndex,
		tryCount,
	)
	if err != nil {
		log.Errorf("Could not create a campaign call. err: %v", err)
		return false, err
	}
	log.WithField("campaigncall", campaignCall).Debugf("Created a new campaigncall. campaigncall_id: %s", campaignCall.ID)

	// create a activeflow
	activeflow, err := h.reqHandler.FMV1ActiveflowCreate(ctx, campaignCall.ActiveflowID, campaignCall.FlowID, activeflow.ReferenceTypeNone, uuid.Nil)
	if err != nil {
		log.Errorf("Could not create an activeflow. err: %v", err)
		return false, err
	}
	log.WithField("activeflow", activeflow).Debugf("Created a new activeflow. activeflow_id: %s", activeflow.ID)

	// execute the activeflow
	if errActiveflow := h.reqHandler.FMV1ActiveflowExecute(ctx, activeflow.ID); errActiveflow != nil {
		log.Errorf("Could not execute the created")
		return false, errActiveflow
	}

	return true, nil
}

// isDialable returns true if a given campaign is dial-able
func (h *campaignHandler) isDialable(ctx context.Context, campaignID, queueID uuid.UUID, serviceLevel int) bool {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "isDialable",
			"id":   campaignID,
		})
	log.Debug("Checking the campaign is dial-able.")

	if queueID == uuid.Nil {
		// the campaign has no queue_id.
		return true
	}

	// get available agents
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

	// calculate the capacity
	dialingLen := len(dialings)
	agentCapacity := (len(agents) * serviceLevel) / 100.0

	if int(agentCapacity) <= dialingLen {
		// currerntly the campaign has enough number of dialings already
		return false
	}
	log.Debugf("The campaign is dialable. agent_capacity: %d, dialing_len: %d", int(agentCapacity), dialingLen)

	return true
}
