package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// CampaignCreate is a service handler for campaign creation.
func (h *serviceHandler) CampaignCreate(
	u *cscustomer.Customer,
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
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignCreate",
		"customer_id": u.ID,
		"name":        name,
	})

	log.Debug("Creating a new campaign.")
	tmp, err := h.reqHandler.CAV1CampaignCreate(
		ctx,
		uuid.Nil,
		u.ID,
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
func (h *serviceHandler) CampaignGetsByCustomerID(u *cscustomer.Customer, size uint64, token string) ([]*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignGetsByCustomerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a campaigns.")

	if token == "" {
		token = getCurTime()
	}

	// get campaigns
	campaigns, err := h.reqHandler.CAV1CampaignGetsByCustomerID(ctx, u.ID, token, size)
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
func (h *serviceHandler) CampaignGet(u *cscustomer.Customer, id uuid.UUID) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Getting an campaign.")

	// get campaign
	tmp, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Errorf("The customer has no permission for this campaign. customer: %s, campaign_customer: %s", u.ID, tmp.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignDelete deletes the campaign.
func (h *serviceHandler) CampaignDelete(u *cscustomer.Customer, id uuid.UUID) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Deleting a campaign.")

	// get campaign
	f, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != f.CustomerID {
		log.Errorf("The customer has no permission for this flow. customer: %s, flow_customer: %s", u.ID, f.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateBasicInfo updates the campaign's basic info.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateBasicInfo(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	tmpCampaign, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpCampaign.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateStatus updates the campaign's status.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateStatus(u *cscustomer.Customer, id uuid.UUID, status cacampaign.Status) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateStatus",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	tmpCampaign, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpCampaign.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignUpdateStatus(ctx, id, status)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateServiceLevel updates the campaign's service level.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateServiceLevel(u *cscustomer.Customer, id uuid.UUID, serviceLevel int) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateServiceLevel",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	tmpCampaign, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpCampaign.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignUpdateServiceLevel(ctx, id, serviceLevel)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateActions updates the campaign's actions.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateActions(u *cscustomer.Customer, id uuid.UUID, actions []fmaction.Action) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateActions",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	tmpCampaign, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpCampaign.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignUpdateActions(ctx, id, actions)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateResourceInfo updates the campaign's resource_info.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateResourceInfo(u *cscustomer.Customer, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateResourceInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	tmpCampaign, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpCampaign.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignUpdateResourceInfo(ctx, id, outplanID, outdialID, queueID)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaignUpdateNextCampaignID updates the campaign's next_campaign_id.
// It returns updated campaign if it succeed.
func (h *serviceHandler) CampaignUpdateNextCampaignID(u *cscustomer.Customer, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaignUpdateNextCampaignID",
		"customer_id": u.ID,
		"username":    u.Username,
		"campaign_id": id,
	})
	log.Debug("Updating an campaign.")

	// get campaign
	tmpCampaign, err := h.reqHandler.CAV1CampaignGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpCampaign.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CAV1CampaignUpdateNextCampaignID(ctx, id, nextCampaignID)
	if err != nil {
		logrus.Errorf("Could not update the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
