package campaignhandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

// Create creates a new campaign
func (h *campaignHandler) Create(
	ctx context.Context,
	id uuid.UUID,

	customerID uuid.UUID,
	campaignType campaign.Type,
	name string,
	detail string,

	serviceLevel int,
	endHandle campaign.EndHandle,

	flowID uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Create",
		"id":               id,
		"customer_id":      customerID,
		"campaign_type":    campaignType,
		"name":             name,
		"detail":           detail,
		"service_level":    serviceLevel,
		"end_handle":       endHandle,
		"flow_id":          flowID,
		"outplan_id":       outplanID,
		"outdial_id":       outdialID,
		"queue_id":         queueID,
		"next_campaign_id": nextCampaignID,
	})

	if id == uuid.Nil {
		id = h.util.UUIDCreate()
		log = log.WithField("id", id)
	}

	// validate
	if !h.validateResources(ctx, id, flowID, outplanID, outdialID, queueID, nextCampaignID) {
		log.Errorf("Could not pass the resource validation. flow_id: %s, outplan_id: %s, outdial: %s, queue_id: %s, next_campaign_id: %s", flowID, outplanID, outdialID, queueID, nextCampaignID)
		return nil, fmt.Errorf("could not pass the resource validation")
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
		FlowID:         flowID,
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
	if !h.updateResources(ctx, res.ID, res.OutdialID) {
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
	log := logrus.WithFields(logrus.Fields{
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
	log := logrus.WithFields(logrus.Fields{
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
func (h *campaignHandler) UpdateResourceInfo(
	ctx context.Context,
	id uuid.UUID,
	flowID uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateResourceInfo",
		"campaign_id": id,
		"flow_id":     flowID,
		"outplan_id":  outplanID,
		"outdial_id":  outdialID,
		"queue_id":    queueID,
	})
	log.Debug("Updating campaign basic info.")

	if !h.validateResources(ctx, id, flowID, outplanID, outdialID, queueID, nextCampaignID) {
		log.Errorf("Could not pass the resource validation. outplan_id: %s, outdial_id: %s, queue_id: %s, nex_campaign_id: %s",
			outplanID, outdialID, queueID, uuid.Nil)
		return nil, fmt.Errorf("could not pass the resource validation")
	}

	if err := h.db.CampaignUpdateResourceInfo(ctx, id, flowID, outplanID, outdialID, queueID, nextCampaignID); err != nil {
		log.Errorf("Could not update campaign. err: %v", err)
		return nil, err
	}

	// get updated campaign
	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign. err: %v", err)
		return nil, err
	}
	log.WithField("campaign", res).Debugf("Updated campaign. campaign_id: %s", id)

	h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, campaign.EventTypeCampaignUpdated, res)

	// update resources
	if !h.updateResources(ctx, res.ID, res.OutdialID) {
		log.Errorf("Could not update the resources")
		return nil, fmt.Errorf("could not update the resources")
	}

	return res, nil
}

// UpdateNextCampaignID updates campaign's next_campaign_id info
func (h *campaignHandler) UpdateNextCampaignID(ctx context.Context, id, nextCampaignID uuid.UUID) (*campaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "UpdateNextCampaignID",
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

// validateResources returns
func (h *campaignHandler) validateResources(
	ctx context.Context,
	id uuid.UUID,
	flowID uuid.UUID,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":             "validateResources",
		"id":               id,
		"flow_id":          flowID,
		"outplan_id":       outplanID,
		"outdial_id":       outdialID,
		"queue_id":         queueID,
		"next_campaign_id": nextCampaignID,
	})

	// flow id
	if !h.isValidFlowID(ctx, flowID) {
		log.Debugf("The flow id is not valid. flow_id: %s", flowID)
		return false
	}

	// outplan id
	if !h.isValidOutplanID(ctx, outplanID) {
		log.Debugf("The outplan id is not valid. outplan_id: %s", outplanID)
		return false
	}

	// outdial id
	if !h.isValidOutdialID(ctx, id, outdialID) {
		log.Debugf("The outdial id is not valid. outplan_id: %s", outplanID)
		return false
	}

	// queue id
	if !h.isValidQueueID(ctx, queueID) {
		log.Debugf("The queue id is not valid. queue_id: %s", queueID)
		return false
	}

	// next campaign id
	if !h.isValidNextCampaignID(ctx, nextCampaignID) {
		log.Debugf("The next campaign id is not valid. next_campaign_id: %s", nextCampaignID)
		return false
	}

	return true
}

// isValidOutdialID returns true if the given outdial id is valid
func (h *campaignHandler) isValidOutdialID(ctx context.Context, id uuid.UUID, outdialID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "isValidOutdialID",
		"campaign_id": id,
		"outdial_id":  outdialID,
	})

	if outdialID == uuid.Nil {
		return true
	}

	// get outdial
	od, err := h.reqHandler.OutdialV1OutdialGet(ctx, outdialID)
	if err != nil {
		log.Errorf("Could not get outdial info. err: %v", err)
		return false
	}
	log.WithField("outdial", od).Debugf("Checking outdial info. outdial_id: %s", od.ID)

	if od.CampaignID != uuid.Nil && od.CampaignID != id {
		log.Debugf("The outdial is used by other campaign already. campaign_id: %s", od.CampaignID)
		return false
	}

	if od.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The outdial is already deleted.")
		return false
	}

	return true
}

// isValidFlowID returns true if the flow id is valid.
func (h *campaignHandler) isValidFlowID(ctx context.Context, flowID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":    "isValidFlowID",
		"flow_id": flowID,
	})

	if flowID == uuid.Nil {
		return true
	}

	// get flow
	f, err := h.reqHandler.FlowV1FlowGet(ctx, flowID)
	if err != nil {
		log.Errorf("Could not get outdial info. err: %v", err)
		return false
	}
	log.WithField("flow", f).Debugf("Checking flow info. flow_id: %s", f.ID)

	if f.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The outdial is already deleted.")
		return false
	}

	return true
}

// isValidOutplanID returns true if the outplan id is valid.
func (h *campaignHandler) isValidOutplanID(ctx context.Context, outplanID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":       "isValidOutplanID",
		"outplan_id": outplanID,
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

	if op.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The outdial is already deleted.")
		return false
	}

	return true
}

// isValidQueueID returns true if the queue id is valid for queue id.
func (h *campaignHandler) isValidQueueID(ctx context.Context, queueID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":     "isValidQueueID",
		"queue_id": queueID,
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

	if q.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The queue is already deleted.")
		return false
	}

	return true
}

// isValidNextCampaignID returns true if the given campaign id is valid for next campaign id
func (h *campaignHandler) isValidNextCampaignID(ctx context.Context, nextCampaignID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":             "isValidNextCampaignID",
		"next_campaign_id": nextCampaignID,
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

	if c.TMDelete != dbhandler.DefaultTimeStamp {
		log.Debugf("The campaign is already deleted.")
		return false
	}

	return true
}

func (h *campaignHandler) updateResources(ctx context.Context, id uuid.UUID, outdialID uuid.UUID) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "updateResources",
		"campaign_id": id,
		"outdial_id":  outdialID,
	})

	// outdial id
	if outdialID != uuid.Nil {
		_, errUpdate := h.reqHandler.OutdialV1OutdialUpdateCampaignID(ctx, outdialID, id)
		if errUpdate != nil {
			log.Errorf("Could not update the campaign id to the outdial. err: %v", errUpdate)
			return false
		}
	}

	return true
}
