package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
)

// campaigncallGet validates the campaigncall's ownership and returns the campaigncall info.
func (h *serviceHandler) campaigncallGet(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaigncallGet",
		"customer_id":     a.CustomerID,
		"campaigncall_id": campaigncallID,
	})

	// send request
	res, err := h.reqHandler.CampaignV1CampaigncallGet(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not get the campaign info. err: %v", err)
		return nil, err
	}
	log.WithField("campaign", res).Debug("Received result.")

	return res, nil
}

// CampaigncallGets gets the list of campaigncalls.
// It returns list of campaigncalls if it succeed.
func (h *serviceHandler) CampaigncallGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaigncallGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting campaigncalls.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get campaigncalls
	tmps, err := h.reqHandler.CampaignV1CampaigncallGets(ctx, a.CustomerID, token, size)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigns info. err: %v", err)
	}

	// create result
	res := []*cacampaigncall.WebhookMessage{}
	for _, f := range tmps {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// CampaigncallGetsByCampaignID gets the list of campaigncalls of the given campaign id.
// It returns list of campaigncalls if it succeed.
func (h *serviceHandler) CampaigncallGetsByCampaignID(ctx context.Context, a *amagent.Agent, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaigncallGetsByCampaignID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting campaigncalls.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get campaign
	c, err := h.campaignGet(ctx, a, campaignID)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get campaigncalls
	ccs, err := h.reqHandler.CampaignV1CampaigncallGetsByCampaignID(ctx, campaignID, token, size)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigns info. err: %v", err)
	}

	// create result
	res := []*cacampaigncall.WebhookMessage{}
	for _, f := range ccs {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// CampaigncallGet gets the campaigncall of the given id.
// It returns campaigncall if it succeed.
func (h *serviceHandler) CampaigncallGet(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "CampaigncallGet",
		"customer_id":     a.CustomerID,
		"username":        a.Username,
		"campaigncall_id": campaigncallID,
	})
	log.Debug("Getting campaigncall.")

	tmp, err := h.campaigncallGet(ctx, a, campaigncallID)
	if err != nil {
		log.Errorf("Could not get campaigncall info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaigncall info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionAll) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaigncallDelete deletes the campaigncall.
func (h *serviceHandler) CampaigncallDelete(ctx context.Context, a *amagent.Agent, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "CampaigncallDelete",
		"agent":           a,
		"campaigncall_id": campaigncallID,
	})
	log.Debug("Deleting a campaigncall.")

	// get campaign
	c, err := h.campaigncallGet(ctx, a, campaigncallID)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find campaign info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1CampaigncallDelete(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not delete the campaign call. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
