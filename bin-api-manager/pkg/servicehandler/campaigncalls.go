package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// campaigncallGet validates the campaigncall's ownership and returns the campaigncall info.
func (h *serviceHandler) campaigncallGet(ctx context.Context, campaigncallID uuid.UUID) (*cacampaigncall.Campaigncall, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "campaigncallGet",
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
func (h *serviceHandler) CampaigncallList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaigncallGets",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting campaigncalls.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// get campaigncalls
	filters := map[cacampaigncall.Field]any{
		cacampaigncall.FieldCustomerID: a.CustomerID,
	}
	tmps, err := h.reqHandler.CampaignV1CampaigncallList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find campaigns info", err)
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
func (h *serviceHandler) CampaigncallGetsByCampaignID(ctx context.Context, a *auth.AuthIdentity, campaignID uuid.UUID, size uint64, token string) ([]*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CampaigncallGetsByCampaignID",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting campaigncalls.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get campaign
	c, err := h.campaignGet(ctx, campaignID)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find campaign info", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// get campaigncalls
	filters := map[cacampaigncall.Field]any{
		cacampaigncall.FieldCampaignID: campaignID,
	}
	ccs, err := h.reqHandler.CampaignV1CampaigncallList(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get campaigns info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find campaigns info", err)
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
func (h *serviceHandler) CampaigncallGet(ctx context.Context, a *auth.AuthIdentity, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "CampaigncallGet",
		"customer_id":     a.CustomerID,
		"username":        a.DisplayName(),
		"campaigncall_id": campaigncallID,
	})
	log.Debug("Getting campaigncall.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	tmp, err := h.campaigncallGet(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not get campaigncall info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find campaigncall info", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CampaigncallDelete deletes the campaigncall.
func (h *serviceHandler) CampaigncallDelete(ctx context.Context, a *auth.AuthIdentity, campaigncallID uuid.UUID) (*cacampaigncall.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "CampaigncallDelete",
		"agent":           a,
		"campaigncall_id": campaigncallID,
	})
	log.Debug("Deleting a campaigncall.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// get campaign
	c, err := h.campaigncallGet(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not get campaign info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("%w: could not find campaign info", err)
	}

	if !h.hasPermission(ctx, a, c.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.reqHandler.CampaignV1CampaigncallDelete(ctx, campaigncallID)
	if err != nil {
		log.Errorf("Could not delete the campaign call. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
