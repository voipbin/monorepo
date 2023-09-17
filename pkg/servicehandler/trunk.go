package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
	rmtrunk "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
)

// trunkGet validates the trunk's ownership and returns the trunk info.
func (h *serviceHandler) trunkGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmtrunk.Trunk, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "trunkGet",
		"customer_id": u.ID,
		"domain_id":   id,
	})

	// send request
	res, err := h.reqHandler.RegistrarV1TrunkGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the trunk info. err: %v", err)
		return nil, err
	}
	log.WithField("trunk", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission for this trunk.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// TrunkCreate is a service handler for trunk creation.
func (h *serviceHandler) TrunkCreate(ctx context.Context, u *cscustomer.Customer, name string, detail string, domainName string, authTypes []rmtrunk.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkCreate",
		"customer_id": u.ID,
		"domain_name": domainName,
		"name":        name,
	})
	log.Debug("Creating a new trunk.")

	tmp, err := h.reqHandler.RegistrarV1TrunkCreate(ctx, u.ID, name, detail, domainName, authTypes, username, password, allowedIPs)
	if err != nil {
		log.Errorf("Could not create a new trunk. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TrunkDelete deletes the trunk of the given id.
func (h *serviceHandler) TrunkDelete(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkDelete",
		"customer_id": u.ID,
		"username":    u.Username,
		"trunk_id":    id,
	})
	log.Debug("Deleting the domain.")

	_, err := h.trunkGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get the domain info. err: %v", err)
		return nil, fmt.Errorf("could not get domain info. err: %v", err)
	}

	// delete
	tmp, err := h.reqHandler.RegistrarV1TrunkDelete(ctx, id)
	if err != nil {
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TrunkGet gets the trunk of the given id.
// It returns trunk if it succeed.
func (h *serviceHandler) TrunkGet(ctx context.Context, u *cscustomer.Customer, id uuid.UUID) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkGet",
		"customer_id": u.ID,
		"username":    u.Username,
		"domain_id":   id,
	})
	log.Debug("Getting a trunk.")

	// get trunk
	tmp, err := h.trunkGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get trunk info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not get trunk info. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// TrunkGets gets the list of trunks of the given customer id.
// It returns list of trunks if it succeed.
func (h *serviceHandler) TrunkGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"fucn":        "TrunkGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting a trunks.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get tmps
	tmps, err := h.reqHandler.RegistrarV1TrunkGetsByCustomerID(ctx, u.ID, token, size)
	if err != nil {
		log.Errorf("Could not get trunks info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find trunks info. err: %v", err)
	}

	// create result
	res := []*rmtrunk.WebhookMessage{}
	for _, d := range tmps {
		tmp := d.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

// TrunkUpdateBasicInfo updates the trunk info.
// It returns updated trunk if it succeed.
func (h *serviceHandler) TrunkUpdateBasicInfo(ctx context.Context, u *cscustomer.Customer, id uuid.UUID, name string, detail string, authTypes []rmtrunk.AuthType, username string, password string, allowedIPs []string) (*rmtrunk.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "TrunkUpdateBasicInfo",
		"customer_id": u.ID,
		"username":    u.Username,
		"trunk_id":    id,
	})
	log.Debug("Updating a domain.")

	// get
	_, err := h.trunkGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not get domain info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find domain info. err: %v", err)
	}

	// update
	tmp, err := h.reqHandler.RegistrarV1TrunkUpdateBasicInfo(ctx, id, name, detail, authTypes, username, password, allowedIPs)
	if err != nil {
		logrus.Errorf("Could not update the trunk. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
