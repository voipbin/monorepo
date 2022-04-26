package campaignhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	omoutdialtarget "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// Execute executes the campaign.
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
		log.Errorf("Could not get campaign. Stopping the campaign execution. campaign_id: %s, err: %v", id, err)
		if err := h.updateExecuteStop(ctx, id); err != nil {
			log.Errorf("Could not stop the campaign execute. err: %v", err)
		}

		return
	}

	// check the campaign's status
	if c.Status != campaign.StatusRun {
		log.WithField("campaign", c).Infof("The campaign status is not run. Stopping the campaign execution. campaign_id: %s, status: %s", c.ID, c.Status)
		if err := h.updateExecuteStop(ctx, id); err != nil {
			log.Errorf("Could not stop the campaign execute. err: %v", err)
		}
		return
	}

	// check the campaign is dial-able
	if !h.isDialable(ctx, c.ID, c.QueueID, c.ServiceLevel) {
		log.Debugf("Campaign is not dialable now.")

		// send an execute request with 5 seconds of delay
		if errExecute := h.reqHandler.CAV1CampaignExecute(ctx, id, 5000); errExecute != nil {
			log.Errorf("Could not execute the campaign. Stopping the campaign execution.")

			if errUpdate := h.updateExecuteStop(ctx, id); errUpdate != nil {
				log.Errorf("Could not stop the campaign execute. err: %v", errUpdate)
			}
		}
		return
	}

	// get outplan
	p, err := h.outplanHandler.Get(ctx, c.OutplanID)
	if err != nil {
		log.Errorf("Could not get outplan. Stopping the campaign execution. err: %v", err)

		if err := h.updateExecuteStop(ctx, id); err != nil {
			log.Errorf("Could not stop the campaign execute. err: %v", err)
		}
		return
	}

	// get destination
	target, destination, destinationIndex, tryCount, err := h.getDestination(ctx, c, p)
	if err != nil {
		log.Errorf("Could not get outdial target destination. Stopping the campaign. err: %v", err)
		if err := h.updateExecuteStop(ctx, id); err != nil {
			log.Errorf("Could not stop the campaign execute. err: %v", err)
		}
		return
	}
	if target == nil {
		// no more target left.
		if c.EndHandle == campaign.EndHandleStop {
			log.Infof("The campaign has no outdial target left and end handle is stop. Stopping the campaign execution. campaign_id: %s", id)
			if err := h.updateExecuteStop(ctx, id); err != nil {
				log.Errorf("Could not stop the campaign execute. err: %v", err)
			}
			return
		}

		// the endhandle is not stop. continue the campaign execution with 5 seconds of delay.
		// send a campaign execute request with 5 seconds of delay
		_ = h.reqHandler.CAV1CampaignExecute(ctx, id, 5000)
		return
	}

	var cc *campaigncall.Campaigncall
	switch c.Type {
	case campaign.TypeCall:
		cc, err = h.executeCall(ctx, c, p, target, destination, destinationIndex, tryCount)
	case campaign.TypeFlow:
		cc, err = h.executeFlow(ctx, c, p, target, destination, destinationIndex, tryCount)
	}
	if err != nil {
		log.Infof("Could not execute the campaign correctly. Stopping campaign execution. err: %v", err)
		if err := h.updateExecuteStop(ctx, id); err != nil {
			log.Errorf("Could not stop the campaign execute. err: %v", err)
		}
		return
	}
	log.WithField("camapaigncall", cc).Debugf("Created a new campaigncall. campaigncall_id: %s", cc.ID)

	// send a campaignexecute request with 500ms delay
	if errExecute := h.reqHandler.CAV1CampaignExecute(ctx, id, 500); errExecute != nil {
		log.Infof("Could not send the campaign execution correctly. Stopping campaign execution. err: %v", err)
		if errStop := h.updateExecuteStop(ctx, id); errStop != nil {
			log.Errorf("Could not stop the campaign execute. err: %v", errStop)
		}
		return
	}
}

// getDestination returns outdialtarget and target address.
// returns outdialtarget, destination, destinationindex, trycount, error
func (h *campaignHandler) getDestination(
	ctx context.Context,
	c *campaign.Campaign,
	p *outplan.Outplan,
) (*omoutdialtarget.OutdialTarget, *cmaddress.Address, int, int, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "getDestination",
			"campaign_id": c.ID,
		},
	)

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
		log.Errorf("Could not get available outdial target. Stopping the campaign. err: %v", err)

		// send an campaign stop request
		_, _ = h.reqHandler.CAV1CampaignUpdateStatus(ctx, c.ID, campaign.StatusStop)
		return nil, nil, 0, 0, err
	}

	if len(targets) == 0 {
		return nil, nil, 0, 0, nil
	}

	// get target destination
	target := targets[0]
	destination, destinationIndex, tryCount := h.getTargetDestination(ctx, &targets[0], p)
	if destination == nil {
		log.WithField("target", target).Error("Something was wrong. Could not find target destination.")
		return nil, nil, 0, 0, err
	}

	return &target, destination, destinationIndex, tryCount, nil
}

// executeCall handles call type of campaigncall's exeucte.
// it checks campaign's available outdial target and make a call.
func (h *campaignHandler) executeCall(
	ctx context.Context,
	c *campaign.Campaign,
	p *outplan.Outplan,
	target *omoutdialtarget.OutdialTarget,
	destination *cmaddress.Address,
	destinationIndex int,
	tryCount int,
) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "executeCall",
			"campaign_id": c.ID,
		})
	log.Debug("Execute executeCall.")

	// create call_id
	callID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())

	// create campaigncall
	cc, err := h.campaigncallHandler.Create(
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
		return nil, err
	}
	log.WithField("campaigncall", cc).Debugf("Created a new campaign call. campaigncall_id: %s", cc.ID)

	// create a call
	newCall, err := h.reqHandler.CMV1CallCreateWithID(ctx, callID, c.CustomerID, c.FlowID, activeflowID, uuid.Nil, p.Source, destination)
	if err != nil {
		// update camapaign call to fail
		_, _ = h.campaigncallHandler.Done(ctx, cc.ID, campaigncall.ResultFail)
		log.Errorf("Could not create a call. err: %v", err)
		return nil, err
	}
	log.WithField("call", newCall).Debugf("Created a new call for campaign call. call_id: %s", newCall.ID)

	return cc, nil
}

// executeFlow creates a new campaigncall for referencetype flow.
func (h *campaignHandler) executeFlow(
	ctx context.Context,
	c *campaign.Campaign,
	p *outplan.Outplan,
	target *omoutdialtarget.OutdialTarget,
	destination *cmaddress.Address,
	destinationIndex int,
	tryCount int,
) (*campaigncall.Campaigncall, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "executeFlow",
			"campaign_id": c.ID,
		})
	log.Debug("Execute executeFlow.")

	// create activeflow_id
	activeflowID := uuid.Must(uuid.NewV4())

	// create a campaigncall
	tmpCC, err := h.campaigncallHandler.Create(
		ctx,
		c.CustomerID,
		c.ID,
		c.OutplanID,
		c.OutdialID,
		target.ID,
		c.QueueID,

		activeflowID,
		c.FlowID,

		campaigncall.ReferenceTypeFlow,
		uuid.Nil,

		p.Source,
		destination,
		destinationIndex,
		tryCount,
	)
	if err != nil {
		log.Errorf("Could not create a campaigncall. err: %v", err)
		return nil, err
	}
	log.WithField("campaigncall", tmpCC).Debugf("Created a new campaigncall. campaigncall_id: %s", tmpCC.ID)

	// upate the campaigncall status to progressing
	cc, err := h.campaigncallHandler.Progressing(ctx, tmpCC.ID)
	if err != nil {
		log.Errorf("Could not update the campaigncall status to progressing. err: %v", err)

		_, _ = h.campaigncallHandler.Done(ctx, cc.ID, campaigncall.ResultFail)
		return nil, err
	}

	// create a activeflow
	activeflow, err := h.reqHandler.FMV1ActiveflowCreate(ctx, cc.ActiveflowID, cc.FlowID, activeflow.ReferenceTypeNone, uuid.Nil)
	if err != nil {
		log.Errorf("Could not create an activeflow. err: %v", err)
		_, _ = h.campaigncallHandler.Done(ctx, cc.ID, campaigncall.ResultFail)
		return nil, err
	}
	log.WithField("activeflow", activeflow).Debugf("Created a new activeflow. activeflow_id: %s", activeflow.ID)

	// execute the activeflow
	if errActiveflow := h.reqHandler.FMV1ActiveflowExecute(ctx, activeflow.ID); errActiveflow != nil {
		log.Errorf("Could not execute the created activeflow. err: %s", errActiveflow)
		_, _ = h.campaigncallHandler.Done(ctx, cc.ID, campaigncall.ResultFail)
		return nil, errActiveflow
	}

	return cc, nil
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
