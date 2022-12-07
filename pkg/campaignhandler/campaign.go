package campaignhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
)

// Create creates a new campaign
func (h *campaignHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	campaignType campaign.Type,
	name string,
	detail string,

	actions []fmaction.Action,
	serviceLevel int,
	endHandle campaign.EndHandle,

	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*campaign.Campaign, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Create",
			"customer_id": customerID,
		},
	)

	// create a flow actions
	flowActions, err := h.createFlowActions(ctx, actions, queueID)
	if err != nil {
		log.Errorf("Could not create a flowaction. err: %v", err)
		return nil, err
	}

	// create a flow
	f, err := h.reqHandler.FMV1FlowCreate(ctx, customerID, fmflow.TypeCampaign, "", "", flowActions, true)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", f).Debugf("Created a flow for campaign. flow_id: %s", f.ID)

	if id == uuid.Nil {
		id = uuid.Must(uuid.NewV4())
	}
	t := &campaign.Campaign{
		ID:             id,
		CustomerID:     customerID,
		Type:           campaignType,
		Name:           name,
		Detail:         detail,
		Status:         campaign.StatusStop,
		Execute:        campaign.ExecuteStop,
		ServiceLevel:   serviceLevel,
		EndHandle:      endHandle,
		FlowID:         f.ID,
		Actions:        actions,
		OutplanID:      outplanID,
		OutdialID:      outdialID,
		QueueID:        queueID,
		NextCampaignID: nextCampaignID,
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

	log.WithField("campaign", res).Debugf("Created a new campaign. campaign_id: %s", res.ID)

	return res, nil
}

// Delete deletes the campaign
func (h *campaignHandler) Delete(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "Delete",
			"campaign_id": id,
		})
	log.Debugf("Deleting a campaign. campaign_id: %s", id)

	c, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	if c.Status != campaign.StatusStop {
		log.Errorf("The campaign is not stop. status: %s", c.Status)
		return nil, err
	}
	log.WithField("campaign", c).Debugf("Deleting.")

	if errDelete := h.db.CampaignDelete(ctx, id); errDelete != nil {
		log.Errorf("Could not delete campaign. err: %v", errDelete)
		return nil, errDelete
	}

	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted campaign. err: %v", err)
		return nil, err
	}
	log.WithField("campaign", res).Debugf("Deleted campaign. campaign_id: %s", res.ID)
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignDeleted, res)

	// delete flow
	f, err := h.reqHandler.FMV1FlowDelete(ctx, res.FlowID)
	if err != nil {
		// we got an error here, but we've deleted the campaign already.
		// just write the log only.
		log.Errorf("Could not delete the flow. err: %v", err)
	} else {
		log.WithField("flow", f).Debugf("Deleted campaign flow. flow_id: %s", f.ID)
	}

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
			"func":        "UpdateResourceInfo",
			"campaign_id": id,
			"outplan_id":  outplanID,
			"outdial_id":  outdialID,
			"queue_id":    queueID,
		})
	log.Debug("Updating campaign basic info.")

	if err := h.db.CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, queueID); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	actions, err := h.createFlowActions(ctx, c.Actions, queueID)
	if err != nil {
		log.Errorf("Could not create a flow actions. err: %v", err)
		return nil, err
	}

	// update flow
	f, err := h.reqHandler.FMV1FlowUpdateActions(ctx, c.FlowID, actions)
	if err != nil {
		log.Errorf("Could not update the flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", f).Debugf("Updated flow. campaign_id: %s, flow_id: %s", id, f.ID)

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
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	return res, nil
}

// UpdateActions updates campaign's actions
func (h *campaignHandler) UpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*campaign.Campaign, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":    "UpdateActions",
			"id":      id,
			"actions": actions,
		})
	log.Debug("Updating campaign actions.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	// generate flow actions
	tmpActions, err := h.createFlowActions(ctx, actions, c.QueueID)
	if err != nil {
		log.Errorf("Could not generate actions. err: %v", err)
		return nil, err
	}

	// update flow
	f, err := h.reqHandler.FMV1FlowUpdateActions(ctx, c.FlowID, tmpActions)
	if err != nil {
		log.Errorf("Could not update the actions. err: %v", err)
		return nil, err
	}
	log.WithField("flow", f).Debugf("Updated actions. flow_id: %s", f.ID)

	// update campaign's actions
	if err := h.db.CampaignUpdateActions(ctx, id, actions); err != nil {
		log.Errorf("Could not update campaign service_level. err: %v", err)
		return nil, err
	}

	// get updated info
	res, err := h.db.CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated campaign info. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	return res, nil
}

// createFlowActions creates actions for campaign
func (h *campaignHandler) createFlowActions(ctx context.Context, actions []fmaction.Action, queueID uuid.UUID) ([]fmaction.Action, error) {

	flowActions := actions
	if queueID == uuid.Nil {
		return flowActions, nil
	}

	action := fmaction.Action{
		Type: fmaction.TypeQueueJoin,
	}

	option := fmaction.OptionQueueJoin{
		QueueID: queueID,
	}

	opt, err := json.Marshal(option)
	if err != nil {
		return nil, err
	}

	action.Option = opt
	flowActions = append(flowActions, action)

	return flowActions, nil
}

// updateExecuteStop updates the campaign execute to stop.
// and it checks the campaign's stop-able then stops the campaign if it stop-able.
// otherwise, stopping the campaign.
func (h *campaignHandler) updateExecuteStop(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "updateExecuteStop",
			"campaign_id": id,
		})
	log.Debug("Updating campaign execute.")

	if err := h.db.CampaignUpdateExecute(ctx, id, campaign.ExecuteStop); err != nil {
		log.Errorf("Could not stop the campaign execute. err: %v", err)
		return err
	}

	// update campaign to stop
	c, err := h.campaignStop(ctx, id)
	if err != nil {
		log.Errorf("Could not stopping the campaign. err: %v", err)
		return err
	}
	log.WithField("campaign", c).Debugf("Stopping the campaign. campaign_id: %s", id)

	return nil
}
