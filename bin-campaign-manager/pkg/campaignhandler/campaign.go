package campaignhandler

import (
	"context"
	"encoding/json"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
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
	log := logrus.WithFields(logrus.Fields{
		"func":        "Create",
		"customer_id": customerID,
	})

	if id == uuid.Nil {
		id = h.util.UUIDCreate()
	}

	// validate
	if !h.validateResources(ctx, id, customerID, outplanID, outdialID, queueID, nextCampaignID) {
		log.Errorf("Could not pass the resource validation. outplan_id: %s, outdial: %s, queue_id: %s, next_campaign_id: %s", outplanID, outdialID, queueID, nextCampaignID)
		return nil, fmt.Errorf("could not pass the resource validation")
	}

	// create a flow actions
	flowActions, err := h.createFlowActions(ctx, actions, queueID)
	if err != nil {
		log.Errorf("Could not create a flowaction. err: %v", err)
		return nil, err
	}

	// create a flow
	f, err := h.reqHandler.FlowV1FlowCreate(ctx, customerID, fmflow.TypeCampaign, "", "", flowActions, true)
	if err != nil {
		log.Errorf("Could not create a flow. err: %v", err)
		return nil, err
	}
	log.WithField("flow", f).Debugf("Created a flow for campaign. flow_id: %s", f.ID)

	t := &campaign.Campaign{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
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

	// update resource
	if !h.updateReferencedResources(ctx, res) {
		log.Errorf("Could not update the resources info.")
		return nil, fmt.Errorf("could not update the resources info")
	}

	return res, nil
}

// Delete deletes the campaign
func (h *campaignHandler) Delete(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
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
	f, err := h.reqHandler.FlowV1FlowDelete(ctx, res.FlowID)
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
	log := logrus.WithFields(logrus.Fields{
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
func (h *campaignHandler) UpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	campaignType campaign.Type,
	serviceLevel int,
	endHandle campaign.EndHandle,
) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "UpdateBasicInfo",
		"id":            id,
		"name":          name,
		"detail":        detail,
		"type":          campaignType,
		"service_level": serviceLevel,
		"end_handle":    endHandle,
	})
	log.Debug("Updating campaign basic info.")

	if err := h.db.CampaignUpdateBasicInfo(ctx, id, name, detail, campaignType, serviceLevel, endHandle); err != nil {
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
func (h *campaignHandler) UpdateResourceInfo(
	ctx context.Context,
	id uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "UpdateResourceInfo",
		"campaign_id":      id,
		"outplan_id":       outplanID,
		"outdial_id":       outdialID,
		"queue_id":         queueID,
		"next_campaign_id": nextCampaignID,
	})
	log.Debug("Updating campaign resource info.")

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}

	if !h.validateResources(ctx, id, c.CustomerID, outplanID, outdialID, queueID, nextCampaignID) {
		log.Errorf("Could not pass the resource validation. outplan_id: %s, outdial_id: %s, queue_id: %s, nex_campaign_id: %s",
			outplanID, outdialID, queueID, nextCampaignID)
		return nil, fmt.Errorf("could not pass the resource validation")
	}

	if err := h.db.CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, queueID, nextCampaignID); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	actions, err := h.createFlowActions(ctx, c.Actions, queueID)
	if err != nil {
		log.Errorf("Could not create a flow actions. err: %v", err)
		return nil, err
	}

	// update flow
	f, err := h.reqHandler.FlowV1FlowUpdateActions(ctx, c.FlowID, actions)
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

	// update resources
	if !h.updateReferencedResources(ctx, res) {
		log.Errorf("Could not update the resources")
		return nil, fmt.Errorf("could not update the resources")
	}

	return res, nil
}

// UpdateNextCampaignID updates campaign's next_campaign_id info
func (h *campaignHandler) UpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
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
	log := logrus.WithFields(logrus.Fields{
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
	log := logrus.WithFields(logrus.Fields{
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
	f, err := h.reqHandler.FlowV1FlowUpdateActions(ctx, c.FlowID, tmpActions)
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
	log := logrus.WithFields(logrus.Fields{
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

func (h *campaignHandler) validateResources(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":             "validateResources",
		"id":               id,
		"customer_id":      customerID,
		"outplan_id":       outplanID,
		"outdial_id":       outdialID,
		"queue_id":         queueID,
		"next_campaign_id": nextCampaignID,
	})

	// outplan id
	if !h.isValidOutplanID(ctx, outplanID, customerID) {
		log.Debugf("The outplan id is not valid. outplan_id: %s", outplanID)
		return false
	}

	// outdial id
	if !h.isValidOutdialID(ctx, outdialID, id, customerID) {
		log.Debugf("The outdial id is not valid. outplan_id: %s", outplanID)
		return false
	}

	// queue id
	if !h.isValidQueueID(ctx, queueID, customerID) {
		log.Debugf("The queue id is not valid. queue_id: %s", queueID)
		return false
	}

	// next campaign id
	if !h.isValidNextCampaignID(ctx, nextCampaignID, customerID) {
		log.Debugf("The next campaign id is not valid. next_campaign_id: %s", nextCampaignID)
		return false
	}

	return true
}

// isValidOutdialID returns true if the given outdial id is valid
func (h *campaignHandler) isValidOutdialID(ctx context.Context, outdialID uuid.UUID, campaignID uuid.UUID, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "isValidOutdialID",
		"campaign_id": campaignID,
		"outdial_id":  outdialID,
		"customer_id": customerID,
	})

	if outdialID == uuid.Nil {
		// no outdial id has given. nothing to verify.
		return true
	}

	// get outdial
	od, err := h.reqHandler.OutdialV1OutdialGet(ctx, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info. err: %v", err)
		return false
	}
	log.WithField("outdial", od).Debugf("Checking outdial info. outdial_id: %s", od.ID)

	if od.CustomerID != customerID {
		log.Debugf("The customer id does not match. customer_id: %s", od.CustomerID)
		return false
	}

	if od.CampaignID != uuid.Nil && od.CampaignID != campaignID {
		log.Debugf("The outdial is used by other campaign already. campaign_id: %s", od.CampaignID)
		return false
	}

	if od.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The outdial is already deleted.")
		return false
	}

	return true
}

// isValidOutplanID returns true if the outplan id is valid.
func (h *campaignHandler) isValidOutplanID(ctx context.Context, outplanID uuid.UUID, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "isValidOutplanID",
		"outplan_id":  outplanID,
		"customer_id": customerID,
	})

	if outplanID == uuid.Nil {
		return true
	}

	// get outplan
	op, err := h.outplanHandler.Get(ctx, outplanID)
	if err != nil {
		log.Errorf("Could not get outdial info. err: %v", err)
		return false
	}
	log.WithField("outplan", op).Debugf("Checking outdial info. outplan_id: %s", op.ID)

	if op.CustomerID != customerID {
		log.Debugf("The customer id does not match. customer_id: %s", op.CustomerID)
		return false
	}

	if op.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The outdial is already deleted.")
		return false
	}

	return true
}

// isValidQueueID returns true if the queue id is valid for queue id.
func (h *campaignHandler) isValidQueueID(ctx context.Context, queueID uuid.UUID, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "isValidQueueID",
		"queue_id":    queueID,
		"customer_id": customerID,
	})

	if queueID == uuid.Nil {
		return true
	}

	// get queue
	q, err := h.reqHandler.QueueV1QueueGet(ctx, queueID)
	if err != nil {
		log.Errorf("Could not get outdial info. err: %v", err)
		return false
	}
	log.WithField("queue", q).Debugf("Checking outdial info. queue_id: %s", q.ID)

	if q.CustomerID != customerID {
		log.Debugf("The customer id does not match. customer_id: %s", q.CustomerID)
		return false
	}

	if q.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The queue is already deleted.")
		return false
	}

	return true
}

// isValidNextCampaignID returns true if the given campaign id is valid for next campaign id
func (h *campaignHandler) isValidNextCampaignID(ctx context.Context, nextCampaignID uuid.UUID, customerID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":             "isValidNextCampaignID",
		"next_campaign_id": nextCampaignID,
		"customer_id":      customerID,
	})

	if nextCampaignID == uuid.Nil {
		return true
	}

	c, err := h.Get(ctx, nextCampaignID)
	if err != nil {
		log.Errorf("Could not get campaign info. err: %v", err)
		return false
	}
	log.WithField("campaign", c).Debugf("Checking campaign info. campaign_id: %s", c.ID)

	if c.CustomerID != customerID {
		log.Debugf("The customer id does not match. customer_id: %s", c.CustomerID)
		return false
	}

	if c.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The campaign is already deleted.")
		return false
	}

	return true
}

// updateReferencedResources updates the referenced resources.
// campaign has referenced resources which has to be updated if the campaign info updated.
// this function updates the referenced resource info
// TODO: maybe in the future, this function will be removed because the referenced resource will listen the
// campaign's update notification to updates their resources info.
func (h *campaignHandler) updateReferencedResources(
	ctx context.Context,
	c *campaign.Campaign,
) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":     "updateReferencedResources",
		"campaign": c,
	})

	// outdial id
	if c.OutdialID != uuid.Nil {
		_, errUpdate := h.reqHandler.OutdialV1OutdialUpdateCampaignID(ctx, c.OutdialID, c.ID)
		if errUpdate != nil {
			log.Errorf("Could not update the campaign id to the outdial. err: %v", errUpdate)
			return false
		}
	}

	return true
}
