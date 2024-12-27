package servicehandler

import (
	"context"
	"fmt"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonaddress "monorepo/bin-common-handler/models/address"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// customerGet validates the customer's ownership and returns the customer info.
func (h *serviceHandler) customerGet(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error) {
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

	return res, nil
}

// CustomerCreate validates the customer's ownership and creates a new customer
func (h *serviceHandler) CustomerCreate(
	ctx context.Context,
	a *amagent.Agent,
	username string,
	password string,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"Username": username,
		"Name":     name,
	})
	log.Debug("Creating a new customer.")

	// check permission
	// only project super admin permssion can create a new customer.
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// check agent existence
	filters := map[string]string{
		"deleted":  "false",
		"username": username,
	}
	ags, err := h.AgentGets(ctx, a, 100, "", filters)
	if err != nil {
		log.Errorf("Could not get agent info. err: %v", err)
		return nil, err
	}
	if len(ags) > 0 {
		log.Errorf("The agent username is already exist. username: %s", username)
		return nil, fmt.Errorf("the agent username is already exist")
	}

	// create customer
	cu, err := h.reqHandler.CustomerV1CustomerCreate(ctx, 30000, name, detail, email, phoneNumber, address, webhookMethod, webhookURI)
	if err != nil {
		log.Errorf("Could not create a new customer. err: %v", err)
		return nil, err
	}

	// create admin agent
	ag, err := h.reqHandler.AgentV1AgentCreate(ctx, 30000, cu.ID, username, password, "default admin", "default agent account for admin permission", amagent.RingMethodRingAll, amagent.PermissionCustomerAdmin, []uuid.UUID{}, []commonaddress.Address{})
	if err != nil {
		log.Errorf("Could not create admin agent. err: %v", err)
		return nil, err
	}
	log.WithField("agent", ag).Debugf("Created admin agent info. agent_id: %s", ag.ID)

	// create billing account
	ba, err := h.reqHandler.BillingV1AccountCreate(ctx, cu.ID, "basic billing account", "billing account for default use", bmaccount.PaymentTypePrepaid, bmaccount.PaymentMethodNone)
	if err != nil {
		log.Errorf("Could not create billing account. err: %v", err)
		return nil, err
	}
	log.WithField("billing_account", ba).Debugf("Created billing account. billing_account_id: %s", ba.ID)

	// set default billing account
	cu, err = h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, cu.ID, ba.ID)
	if err != nil {
		log.Errorf("Could not update default billing account. err: %v", err)
		return nil, err
	}
	log.WithField("customer", cu).Debugf("Updated customer info. customer_id: %s", cu.ID)

	res := cu.ConvertWebhookMessage()
	return res, nil
}

// CustomerGet returns customer info of given customerID.
func (h *serviceHandler) CustomerGet(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerGet",
		"customer_id": a.CustomerID,
	})

	tmp, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, tmp.ID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CustomerGets returns list of all customers
func (h *serviceHandler) CustomerGets(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "CustomerGets",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})
	log.Debug("Received request detail.")

	// check permission
	// only project super admin permssion allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if size <= 0 {
		size = 10
	}
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	tmp, err := h.reqHandler.CustomerV1CustomerGets(ctx, token, size, filters)
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
	a *amagent.Agent,
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
		"customer_id":    a.CustomerID,
		"username":       a.Username,
		"name":           name,
		"detail":         detail,
		"email":          email,
		"phone_number":   phoneNumber,
		"address":        address,
		"webhook_method": webhookMethod,
		"webhook_uri":    webhookURI,
	})

	c, err := h.customerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.ID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
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
func (h *serviceHandler) CustomerDelete(ctx context.Context, a *amagent.Agent, customerID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerDelete",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	// check permission
	// only admin permssion allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	_, err := h.customerGet(ctx, customerID)
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

// CustomerUpdateBillingAccountID sends a request to customer-manager
// to update the customer's billing account id.
func (h *serviceHandler) CustomerUpdateBillingAccountID(ctx context.Context, a *amagent.Agent, customerID uuid.UUID, billingAccountID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "CustomerUpdateBillingAccountID",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
	})

	c, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, c.ID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, fmt.Errorf("agent has no permission")
	}

	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Errorf("Could not validate the billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, customerID, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer's permission. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}
