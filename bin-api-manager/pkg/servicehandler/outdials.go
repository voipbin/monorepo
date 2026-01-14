package servicehandler

import (
	"context"
	"fmt"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// outdialGet validates the outdial's ownership and returns the outdial info.
func (h *serviceHandler) outdialGet(ctx context.Context, id uuid.UUID) (*omoutdial.Outdial, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":     "outdialGet",
		"agent_id": id,
	})

	// send request
	res, err := h.reqHandler.OutdialV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the outdial info. err: %v", err)
		return nil, err
	}
	log.WithField("outdial", res).Debug("Received result.")

	return res, nil
}

// OutdialCreate is a service handler for outdial creation.
func (h *serviceHandler) OutdialCreate(ctx context.Context, a *amagent.Agent, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialCreate",
		"customer_id": a.CustomerID,
		"campaign_id": campaignID,
		"name":        name,
	})
	log.Debug("Creating a new outdial.")

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.OutdialV1OutdialCreate(ctx, a.CustomerID, campaignID, name, detail, data)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialGetsByCustomerID gets the list of outdials of the given customer id.
// It returns list of outdials if it succeed.
func (h *serviceHandler) OutdialGetsByCustomerID(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialGetsByCustomerID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a outdials.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get outdials
	filters := map[omoutdial.Field]any{
		omoutdial.FieldCustomerID: a.CustomerID,
	}
	outdials, err := h.reqHandler.OutdialV1OutdialGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get outdials info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdials info. err: %v", err)
	}

	// create result
	res := []*omoutdial.WebhookMessage{}
	for _, f := range outdials {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// OutdialDelete deletes the outdial.
func (h *serviceHandler) OutdialDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  id,
	})
	log.Debug("Deleting a outdial.")

	// get outdial
	od, err := h.outdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.OutdialV1OutdialDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialGet gets the outdial of the given id.
// It returns outdial if it succeed.
func (h *serviceHandler) OutdialGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialGet",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  id,
	})
	log.Debug("Getting an outdial.")

	// get outdial
	tmp, err := h.outdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialUpdateBasicInfo updates the outdial's basic info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateBasicInfo(ctx context.Context, a *amagent.Agent, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  id,
	})
	log.Debug("Updating an outdial.")

	// get outdial
	od, err := h.outdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.OutdialV1OutdialUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialUpdateCampaignID updates the outdial's campaignID info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateCampaignID(ctx context.Context, a *amagent.Agent, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateCampaignID",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  id,
		"campaign_id": campaignID,
	})
	log.Debug("Executing OutdialUpdateCampaignID.")

	// get outdial
	od, err := h.outdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.OutdialV1OutdialUpdateCampaignID(ctx, id, campaignID)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialUpdateData updates the outdial's data info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateData(ctx context.Context, a *amagent.Agent, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateData",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"outdial_id":  id,
		"data":        data,
	})
	log.Debug("Executing OutdialUpdateData.")

	// get outdial
	od, err := h.outdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, od.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	tmp, err := h.reqHandler.OutdialV1OutdialUpdateData(ctx, id, data)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
