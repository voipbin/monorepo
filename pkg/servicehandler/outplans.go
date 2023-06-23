package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// OutplanCreate is a service handler for outplan creation.
func (h *serviceHandler) OutplanCreate(
	ctx context.Context,
	u *cscustomer.Customer,
	name string,
	detail string,
	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanCreate",
		"customer_id": u.ID,
		"name":        name,
	})

	log.Debug("Creating a new outplan.")
	tmp, err := h.reqHandler.CampaignV1OutplanCreate(ctx, u.ID, name, detail, source, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4)
	if err != nil {
		log.Errorf("Could not create a new outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanDelete deletes the outplan.
func (h *serviceHandler) OutplanDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"outplan_id":  id,
	})
	log.Debug("Deleting a outplan.")

	// get outplan
	f, err := h.reqHandler.CampaignV1OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get flow info from the flow-manager. err: %v", err)
		return nil, fmt.Errorf("could not find flow info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != f.CustomerID {
		log.Errorf("The customer has no permission for this flow. customer: %s, flow_customer: %s", u.ID, f.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1OutplanDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanGetsByCustomerID gets the list of outplans of the given customer id.
// It returns list of outplans if it succeed.
func (h *serviceHandler) OutplanGetsByCustomerID(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a outplans.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get outplans
	outplans, err := h.reqHandler.CampaignV1OutplanGetsByCustomerID(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get outplans info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplans info. err: %v", err)
	}

	// create result
	res := []*caoutplan.WebhookMessage{}
	for _, f := range outplans {
		tmp := f.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// OutplanGet gets the outplan of the given id.
// It returns outplan if it succeed.
func (h *serviceHandler) OutplanGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"outplan_id":  id,
	})
	log.Debug("Getting an outplan.")

	// get outplan
	tmp, err := h.reqHandler.CampaignV1OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	// permission check
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmp.CustomerID {
		log.Errorf("The customer has no permission for this outplan. customer: %s, outplan_customer: %s", u.ID, tmp.CustomerID)
		return nil, fmt.Errorf("customer has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanUpdateBasicInfo updates the outplan's basic info.
// It returns updated outplan if it succeed.
func (h *serviceHandler) OutplanUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name, detail string) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"outplan_id":  id,
	})
	log.Debug("Updating an outplan.")

	// get outplan
	tmpOutplan, err := h.reqHandler.CampaignV1OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpOutplan.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1OutplanUpdateBasicInfo(ctx, id, name, detail)
	if err != nil {
		logrus.Errorf("Could not update the outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// OutplanUpdateDialInfo updates the outplan's dial info.
// It returns updated outplan if it succeed.
func (h *serviceHandler) OutplanUpdateDialInfo(
	ctx context.Context,
	u *cscustomer.Customer,
	id uuid.UUID,
	source *commonaddress.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*caoutplan.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "OutplanUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"outplan_id":  id,
	})
	log.Debug("Updating an outplan.")

	// get outplan
	tmpOutplan, err := h.reqHandler.CampaignV1OutplanGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get outplan info from the campaign-manager. err: %v", err)
		return nil, fmt.Errorf("could not find outplan info. err: %v", err)
	}

	// check the ownership
	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != tmpOutplan.CustomerID {
		log.Info("The customer has no permission.")
		return nil, fmt.Errorf("customer has no permission")
	}

	tmp, err := h.reqHandler.CampaignV1OutplanUpdateDialInfo(ctx, id, source, dialTimeout, tryInterval, maxTryCount0, maxTryCount1, maxTryCount2, maxTryCount3, maxTryCount4)
	if err != nil {
		logrus.Errorf("Could not update the outplan. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
