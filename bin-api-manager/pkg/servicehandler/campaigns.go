package servicehandler

import (
	"context"
	"fmt"

	cacampaign "monorepo/bin-campaign-manager/models/campaign"

	fmaction "monorepo/bin-flow-manager/models/action"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// campaignGet validates the campaign's ownership and returns the campaign info.
func (h *serviceHandler) campaignGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cacampaign.Campaign, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "campaignGet",
		"customer_id": a.CustomerID,
		"campaign_id": id,
	})

	// send request
	res, err := h.reqHandler.CampaignV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the campaign info. err: %v", err)
		return nil, err
	}
	log.WithField("campaign", res).Debug("Received result.")

	return res, nil
}

// CampaignCreate is a service handler for campaign creation.
func (h *serviceHandler) CampaignCreate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	campaignType cacampaign.Type,
	serviceLevel int,
	endHandle cacampaign.EndHandle,
	actions []fmaction.Action,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignCreate",
		"customer_id": a.CustomerID,
		"name":        name,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	log.Debug("Creating a new campaign.")
	tmp, err := h.reqHandler.CampaignV1CampaignCreate(
		ctx,
		uuid.Nil,
		a.CustomerID,
		campaignType,
		name,
		detail,
		serviceLevel,
		endHandle,
		actions,
		outplanID,
		outdialID,
		queueID,
		nextCampaignID,
	)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignGetsByCustomerID gets the list of campaigns of the given customer id.
// It returns list of campaigns if it succeed.
func (h *serviceHandler) CampaignGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a campaigns.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get campaigns
	campaigns, err := h.reqHandler.CampaignV1CampaignGetsByCustomerID(ctx, a.CustomerID, token, size)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigns info. err: %v", err)
	}

	// create result
	res := []*cacampaign.WebhookMessage{}
	for _, f := range campaigns {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// CampaignGet gets the campaign of the given id.
// It returns campaign if it succeed.
func (h *serviceHandler) CampaignGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Getting an campaign.")

	// get campaign
	tmp, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignDelete deletes the campaign.
func (h *serviceHandler) CampaignDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Deleting a campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateBasicInfo updates the campaign's basic info.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateBasicInfo(
	ctx context.Context,
	a *amagent.Agent,
	id uuid.UUID,
	name string,
	detail string,
	campaignType cacampaign.Type,
	serviceLevel int,
	endHandle cacampaign.EndHandle,
) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "CampaignUpdateBasicInfo",
		"customer_id":   a.CustomerID,
		"username":      a.Username,
		"campaign_id":   id,
		"name":          name,
		"detail":        detail,
		"type":          campaignType,
		"service_level": serviceLevel,
		"end_handle":    endHandle,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignUpdateBasicInfo(ctx, id, name, detail, campaignType, serviceLevel, endHandle)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateStatus updates the campaign's status.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateStatus(ctx context.Context, a *amagent.Agent, id uuid.UUID, status cacampaign.Status) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateStatus",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignUpdateStatus(ctx, id, status)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateServiceLevel updates the campaign's service level.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateServiceLevel(ctx context.Context, a *amagent.Agent, id uuid.UUID, serviceLevel int) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateServiceLevel",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignUpdateServiceLevel(ctx, id, serviceLevel)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateActions updates the campaign's actions.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateActions(ctx context.Context, a *amagent.Agent, id uuid.UUID, actions []fmaction.Action) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateActions",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignUpdateActions(ctx, id, actions)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateResourceInfo updates the campaign's resource_info.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateResourceInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateResourceInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, queueID, nextCampaignID)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateNextCampaignID updates the campaign's next_campaign_id.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateNextCampaignID(ctx context.Context, a *amagent.Agent, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateNextCampaignID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	c, err := h.campaignGet(ctx, a, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaignUpdateNextCampaignID(ctx, id, nextCampaignID)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
