package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
)

// outdialGet validates the outdial's ownership and returns the outdial info.
func (h *serviceHandler) outdialGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "outdialGet",
			"customer_id": u.ID,
			"agent_id":    id,
		},
	)

	// send request
	tmp, err := h.reqHandler.OutdialV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the outdial info. err: %v", err)
		return nil, err
	}
	log.WithField("outdial", tmp).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Info("The user has no permission for this agent.")
		return nil, fmt.Errorf("user has no permission")
	}

	// create result
	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialCreate is a service handler for outdial creation.
func (h *serviceHandler) OutdialCreate(ctx context.Context, u *cscustomer.Customer, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"campaign_id": campaignID,
		"name":        name,
	})

	log.Debug("Creating a new outdial.")
	tmp, err := h.reqHandler.OutdialV1OutdialCreate(ctx, u.ID, campaignID, name, detail, data)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialGetsByCustomerID gets the list of outdials of the given customer id.
// It returns list of outdials if it succeed.
func (h *serviceHandler) OutdialGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialGetsByCustomerID",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a outdials.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get outdials
	outdials, err := h.reqHandler.OutdialV1OutdialGetsByCustomerID(ctx, u.ID, token, size)
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
func (h *serviceHandler) OutdialDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
	})
	log.Debug("Deleting a outdial.")

	// get outdial
	_, err := h.outdialGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
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
func (h *serviceHandler) OutdialGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
	})
	log.Debug("Getting an outdial.")

	// get outdial
	res, err := h.outdialGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	return res, nil
}

// OutdialUpdateBasicInfo updates the outdial's basic info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
	})
	log.Debug("Updating an outdial.")

	// get outdial
	_, err := h.outdialGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
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
func (h *serviceHandler) OutdialUpdateCampaignID(ctx context.Context, u *cscustomer.Customer, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateCampaignID",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
		"campaign_id": campaignID,
	})
	log.Debug("Executing OutdialUpdateCampaignID.")

	// get outdial
	_, err := h.outdialGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
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
func (h *serviceHandler) OutdialUpdateData(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateData",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
		"data":        data,
	})
	log.Debug("Executing OutdialUpdateData.")

	// get outdial
	_, err := h.outdialGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not get outdial info. err: %v", err)
	}

	tmp, err := h.reqHandler.OutdialV1OutdialUpdateData(ctx, id, data)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
