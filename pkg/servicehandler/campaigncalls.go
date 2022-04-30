package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// campaigncallGet validates the campaigncall's ownership and returns the campaigncall info.
func (h *serviceHandler) campaigncallGet(ctx context.Context, u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "campaigncallGet",
			"customer_id": u.ID,
			"agent_id":    campaigncallID,
		},
	)

	// send request
	tmp, err := h.reqHandler.CAV1CampaigncallGet(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not get the campaign info. err: %v", err)
		return nil, err
	}
	log.WithField("campaign", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaigncallGetsByCampaignID gets the list of campaigncalls of the given campaign id.
// It returns list of campaigncalls if it succeed.
func (h *serviceHandler) CampaigncallGetsByCampaignID(u *cscustomer.Customer, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaigncallGetsByCampaignID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting campaigncalls.")

	if token == "" {
		token = getCurTime()
	}

	// get campaign
	_, err := h.campaignGet(ctx, u, campaignID)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	// get campaigns
	campaigns, err := h.reqHandler.CAV1CampaigncallGetsByCampaignID(ctx, campaignID, token, size)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigns info. err: %v", err)
	}

	// create result
	res := []*cacampaigncall.WebhookMessage{}
	for _, f := range campaigns {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// CampaigncallGet gets the campaigncall of the given id.
// It returns campaigncall if it succeed.
func (h *serviceHandler) CampaigncallGet(u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":            "CampaigncallGet",
		"customer_id":     u.ID,
		"username":        u.Username,
		"campaigncall_id": campaigncallID,
	})
	log.Debug("Getting campaigncall.")

	res, err := h.campaigncallGet(ctx, u, campaigncallID)
	if err != nil {
		log.Errorf("Could not get campaigncall info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigncall info. err: %v", err)
	}

	return res, nil
}

// CampaigncallDelete deletes the campaigncall.
func (h *serviceHandler) CampaigncallDelete(u *cscustomer.Customer, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":            "CampaigncallDelete",
		"customer_id":     u.ID,
		"username":        u.Username,
		"campaigncall_id": campaigncallID,
	})
	log.Debug("Deleting a campaigncall.")

	// get campaign
	_, err := h.campaigncallGet(ctx, u, campaigncallID)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	tmp, err := h.reqHandler.CAV1CampaigncallDelete(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not delete the campaign. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
