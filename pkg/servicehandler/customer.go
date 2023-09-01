package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// customerGet validates the customer's ownership and returns the customer info.
func (h *serviceHandler) customerGet(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "customerGet",
		"customer_id": customerID,
	})

	// send request
	res, err := h.reqHandler.CustomerV1CustomerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get the customer. err: %v", err)
		return nil, err
	}
	log.WithField("customer", res).Debug("Received result.")

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.ID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// CustomerCreate validates the customer's ownership and creates a new customer
func (h *serviceHandler) CustomerCreate(
	ctx context.Context,
	u *cscustomer.Customer,
	username string,
	password string,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
	permissionIDs []uuid.UUID,
) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"Username": username,
		"Name":     name,
	})
	log.Debug("Creating a new customer.")

	// check permission
	// only admin permssion can create a new customer.
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.CustomerV1CustomerCreate(ctx, 30000, username, password, name, detail, email, phoneNumber, address, webhookMethod, webhookURI, permissionIDs)
	if err != nil {
		log.Errorf("Could not create a new customer. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// UserGet returns customer info of given customerID.
func (h *serviceHandler) CustomerGet(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerGet",
		"customer_id": u.ID,
	})

	tmp, err := h.customerGet(ctx, u, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CustomerGets returns list of all customers
func (h *serviceHandler) CustomerGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "CustomerGets",
		"size":  size,
		"token": token,
	})
	log.Debug("Received request detail.")

	// check permission
	// only admin permssion allowed
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	if size <= 0 {
		size = 10
	}
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmp, err := h.reqHandler.CustomerV1CustomerGets(ctx, token, size)
	if err != nil {
		log.Errorf("Could not get customers info. err: %v", err)
		return nil, err
	}

	res := []*cscustomer.WebhookMessage{}
	for _, u := range tmp {
		t := u.ConvertWebhookMessage()
		res = append(res, t)
	}

	return res, nil
}

// CustomerUpdate sends a request to customer-manager
// to update the customer's basic info.
func (h *serviceHandler) CustomerUpdate(
	ctx context.Context,
	u *cscustomer.Customer,
	id uuid.UUID,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "CustomerUpdate",
		"customer_id":    u.ID,
		"username":       u.Username,
		"name":           name,
		"detail":         detail,
		"email":          email,
		"phone_number":   phoneNumber,
		"address":        address,
		"webhook_method": webhookMethod,
		"webhook_uri":    webhookURI,
	})

	_, err := h.customerGet(ctx, u, id)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdate(ctx, id, name, detail, email, phoneNumber, address, webhookMethod, webhookURI)
	if err != nil {
		log.Errorf("Could not update the customer's basic info. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerDelete sends a request to customer-manager
// to delete the customer.
func (h *serviceHandler) CustomerDelete(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerDelete",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	// check permission
	// only admin permssion allowed
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	_, err := h.customerGet(ctx, u, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerDelete(ctx, customerID)
	if err != nil {
		log.Errorf("Could not delete the customer. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerUpdatePassword sends a request to customer-manager
// to update the customer's password.
func (h *serviceHandler) CustomerUpdatePassword(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, password string) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerUpdatePassword",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	_, err := h.customerGet(ctx, u, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdatePassword(ctx, 30000, customerID, password)
	if err != nil {
		log.Infof("Could not update the customer's password. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerUpdatePermissionIDs sends a request to customer-manager
// to update the customer's permission ids.
func (h *serviceHandler) CustomerUpdatePermissionIDs(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, permissionIDs []uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerUpdatePermissionIDs",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	// note. currently, only admin permission allows to update the
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdatePermissionIDs(ctx, customerID, permissionIDs)
	if err != nil {
		log.Errorf("Could not update the customer's permission. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerUpdateBillingAccountID sends a request to customer-manager
// to update the customer's billing account id.
func (h *serviceHandler) CustomerUpdateBillingAccountID(ctx context.Context, u *cscustomer.Customer, customerID uuid.UUID, billingAccountID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "CustomerUpdateBillingAccountID",
		"customer_id":        u.ID,
		"username":           u.Username,
		"billing_account_id": billingAccountID,
	})

	_, err := h.customerGet(ctx, u, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	_, err = h.billingAccountGet(ctx, u, billingAccountID)
	if err != nil {
		log.Errorf("Could not validate the billing account info. err: %v", err)
		return nil, err
	}

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, customerID, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer's permission. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}
