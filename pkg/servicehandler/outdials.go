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

// OutdialCreate is a service handler for outdial creation.
func (h *serviceHandler) OutdialCreate(u *cscustomer.Customer, campaignID uuid.UUID, name, detail, data string) (*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"campaign_id": campaignID,
		"name":        name,
	})

	log.Debug("Creating a new outdial.")
	tmp, err := h.reqHandler.OMV1OutdialCreate(ctx, u.ID, campaignID, name, detail, data)
	if err != nil {
		log.Errorf("Could not create a new flow. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialGets gets the list of outdials of the given customer id.
// It returns list of outdials if it succeed.
func (h *serviceHandler) OutdialGets(u *cscustomer.Customer, size uint64, token string) ([]*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a outdials.")

	if token == "" {
		token = getCurTime()
	}

	// get outdials
	outdials, err := h.reqHandler.OMV1OutdialGetsByCustomerID(ctx, u.ID, token, size)
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
func (h *serviceHandler) OutdialDelete(u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
	})
	log.Debug("Deleting a outdial.")

	// get outdial
	f, err := h.reqHandler.OMV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != f.CustomerID {
		log.Errorf("The customer has no permission for this flow. customer: %s, flow_customer: %s", u.ID, f.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.OMV1OutdialDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialGet gets the outdial of the given id.
// It returns outdial if it succeed.
func (h *serviceHandler) OutdialGet(u *cscustomer.Customer, id uuid.UUID) (*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
	})
	log.Debug("Getting an outdial.")

	// get outdial
	tmp, err := h.reqHandler.OMV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdial info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Errorf("The customer has no permission for this outdial. customer: %s, outdial_customer: %s", u.ID, tmp.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialUpdateBasicInfo updates the outdial's basic info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateBasicInfo(u *cscustomer.Customer, id uuid.UUID, name, detail string) (*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
	})
	log.Debug("Updating an outdial.")

	// get outdial
	tmpOutdial, err := h.reqHandler.OMV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdial info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpOutdial.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.OMV1OutdialUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialUpdateCampaignID updates the outdial's campaignID info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateCampaignID(u *cscustomer.Customer, id, campaignID uuid.UUID) (*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateCampaignID",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
		"campaign_id": campaignID,
	})
	log.Debug("Executing OutdialUpdateCampaignID.")

	// get outdial
	tmpFlow, err := h.reqHandler.OMV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdial info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpFlow.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.OMV1OutdialUpdateCampaignID(ctx, id, campaignID)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutdialUpdateData updates the outdial's data info.
// It returns updated outdial if it succeed.
func (h *serviceHandler) OutdialUpdateData(u *cscustomer.Customer, id uuid.UUID, data string) (*omoutdial.WebhookMessage, error) {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutdialUpdateData",
		"customer_id": u.ID,
		"username":    u.Username,
		"outdial_id":  id,
		"data":        data,
	})
	log.Debug("Executing OutdialUpdateData.")

	// get outdial
	tmpFlow, err := h.reqHandler.OMV1OutdialGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outdial info from the outdial-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outdial info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpFlow.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.OMV1OutdialUpdateData(ctx, id, data)
	if err != nil {
		logrus.Errorf("Could not update the outdial. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
