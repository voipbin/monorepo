package servicehandler

import (
	"context"
	"fmt"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	amagent "monorepo/bin-agent-manager/models/agent"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

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
	a *auth.AuthIdentity,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"agent": a,
		"email": email,
	})
	log.Debug("Creating a new customer.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// check permission
	// only project super admin permssion can create a new customer.
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	// create customer
	res, err := h.reqHandler.CustomerV1CustomerCreate(ctx, 30000, name, detail, email, phoneNumber, address, webhookMethod, webhookURI)
	if err != nil {
		log.Errorf("Could not create a new customer. err: %v", err)
		return nil, err
	}

	// Auto-create empty OutboundConfig for the new customer.
	// Empty whitelist blocks all PSTN calls until the customer explicitly configures it.
	// Fire-and-forget: OutboundConfig failure does not block customer creation.
	if _, cfgErr := h.reqHandler.CallV1OutboundConfigCreate(ctx, res.ID, &cmoutboundconfig.UpdateRequest{}); cfgErr != nil {
		log.Warnf("Could not auto-create OutboundConfig for new customer. customer_id: %s, err: %v", res.ID, cfgErr)
	}

	return res, nil
}

// CustomerGet returns customer info of given customerID.
// Requires ProjectSuperAdmin permission.
func (h *serviceHandler) CustomerGet(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerGet",
		"customer_id": customerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", tmp).Debugf("Retrieved customer info. customer_id: %s", tmp.ID)

	return tmp, nil
}

// CustomerSelfGet returns the authenticated agent's own customer info.
// Requires CustomerAdmin or CustomerManager permission.
func (h *serviceHandler) CustomerSelfGet(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfGet",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	tmp, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", tmp).Debugf("Retrieved customer info. customer_id: %s", tmp.ID)

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// CustomerGets returns list of all customers
func (h *serviceHandler) CustomerList(ctx context.Context, a *auth.AuthIdentity, size uint64, token string, filters map[string]string) ([]*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "CustomerGets",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})
	log.Debug("Received request detail.")

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// check permission
	// only project super admin permssion allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	if size <= 0 {
		size = 10
	}
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertCustomerFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.CustomerV1CustomerList(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get customers info. err: %v", err)
		return nil, err
	}

	res := make([]*cscustomer.Customer, len(tmps))
	for i := range tmps {
		res[i] = &tmps[i]
	}

	return res, nil
}

// CustomerUpdate sends a request to customer-manager
// to update the customer's basic info.
// Requires ProjectSuperAdmin permission.
func (h *serviceHandler) CustomerUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	id uuid.UUID,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "CustomerUpdate",
		"customer_id":    id,
		"username":       a.DisplayName(),
		"name":           name,
		"detail":         detail,
		"email":          email,
		"phone_number":   phoneNumber,
		"address":        address,
		"webhook_method": webhookMethod,
		"webhook_uri":    webhookURI,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	c, err := h.customerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdate(ctx, id, name, detail, email, phoneNumber, address, webhookMethod, webhookURI)
	if err != nil {
		log.Errorf("Could not update the customer's basic info. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerSelfUpdate updates the authenticated agent's own customer info.
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfUpdate(
	ctx context.Context,
	a *auth.AuthIdentity,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfUpdate",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdate(ctx, a.CustomerID, name, detail, email, phoneNumber, address, webhookMethod, webhookURI)
	if err != nil {
		log.Errorf("Could not update the customer's basic info. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerDelete sends a request to customer-manager
// to delete the customer.
func (h *serviceHandler) CustomerDelete(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// check permission
	// only admin permssion allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
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

	return res, nil
}

// CustomerFreeze sends a request to customer-manager
// to freeze the customer account (schedule deletion).
func (h *serviceHandler) CustomerFreeze(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerFreeze",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// check permission
	// only admin permission allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerFreeze(ctx, customerID)
	if err != nil {
		log.Errorf("Could not freeze the customer. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerRecover sends a request to customer-manager
// to recover the customer account (cancel scheduled deletion).
func (h *serviceHandler) CustomerRecover(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerRecover",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// check permission
	// only admin permission allowed
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission for this agent.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerRecover(ctx, customerID)
	if err != nil {
		log.Errorf("Could not recover the customer. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerSelfFreeze handles self-service account freeze (schedule deletion).
// Requires CustomerAdmin permission and operates on the agent's own customer account.
func (h *serviceHandler) CustomerSelfFreeze(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfFreeze",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerFreeze(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not freeze the customer. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerSelfFreezeAndDelete handles self-service immediate account deletion.
// Freezes and immediately deletes the account (skips 30-day grace period).
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfFreezeAndDelete(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfFreezeAndDelete",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerFreezeAndDelete(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not freeze and delete the customer. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerSelfRecover handles self-service account recovery (cancel scheduled deletion).
// Requires CustomerAdmin permission and operates on the agent's own customer account.
func (h *serviceHandler) CustomerSelfRecover(ctx context.Context, a *auth.AuthIdentity) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfRecover",
		"customer_id": a.CustomerID,
		"username":    a.DisplayName(),
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	res, err := h.reqHandler.CustomerV1CustomerRecover(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not recover the customer. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerUpdateBillingAccountID sends a request to customer-manager
// to update the customer's billing account id.
// Requires ProjectSuperAdmin permission.
func (h *serviceHandler) CustomerUpdateBillingAccountID(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, billingAccountID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "CustomerUpdateBillingAccountID",
		"customer_id":        customerID,
		"billing_account_id": billingAccountID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	_, err = h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Errorf("Could not validate the billing account info. err: %v", err)
		return nil, err
	}

	// send request
	res, err := h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, customerID, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer's billing account. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerUpdateDefaultOutgoingSourceNumberID sends a request to customer-manager
// to update the customer's default outgoing source number id.
// Requires ProjectSuperAdmin permission.
func (h *serviceHandler) CustomerUpdateDefaultOutgoingSourceNumberID(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, defaultOutgoingSourceNumberID uuid.UUID) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                              "CustomerUpdateDefaultOutgoingSourceNumberID",
		"customer_id":                       customerID,
		"default_outgoing_source_number_id": defaultOutgoingSourceNumberID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	_, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}

	// validate the number exists and belongs to this customer
	num, err := h.numberGet(ctx, defaultOutgoingSourceNumberID)
	if err != nil {
		log.Errorf("Could not validate the number info. err: %v", err)
		return nil, err
	}
	log.WithField("number", num).Debugf("Retrieved number info. number_id: %s", num.ID)
	if num.CustomerID != customerID {
		log.Infof("The number does not belong to this customer. number_customer_id: %s", num.CustomerID)
		return nil, fmt.Errorf("%w: the number does not belong to this customer", serviceerrors.ErrPermissionDenied)
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateDefaultOutgoingSourceNumberID(ctx, customerID, defaultOutgoingSourceNumberID)
	if err != nil {
		log.Errorf("Could not update the customer's default outgoing source number id. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerUpdateMetadata updates the customer's internal metadata.
// Requires ProjectSuperAdmin permission.
func (h *serviceHandler) CustomerUpdateMetadata(ctx context.Context, a *auth.AuthIdentity, customerID uuid.UUID, metadata cscustomer.Metadata) (*cscustomer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerUpdateMetadata",
		"customer_id": customerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	c, err := h.customerGet(ctx, customerID)
	if err != nil {
		log.Errorf("Could not validate the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	res, err := h.reqHandler.CustomerV1CustomerUpdateMetadata(ctx, customerID, metadata)
	if err != nil {
		log.Errorf("Could not update the customer's metadata. err: %v", err)
		return nil, err
	}

	return res, nil
}

// CustomerSelfUpdateBillingAccountID updates the authenticated agent's own customer's billing account ID.
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfUpdateBillingAccountID(ctx context.Context, a *auth.AuthIdentity, billingAccountID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "CustomerSelfUpdateBillingAccountID",
		"customer_id":        a.CustomerID,
		"billing_account_id": billingAccountID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Errorf("Could not validate the billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, a.CustomerID, billingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer's billing account. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerSelfUpdateDefaultOutgoingSourceNumberID updates the authenticated agent's own customer's default outgoing source number id.
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfUpdateDefaultOutgoingSourceNumberID(ctx context.Context, a *auth.AuthIdentity, defaultOutgoingSourceNumberID uuid.UUID) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                              "CustomerSelfUpdateDefaultOutgoingSourceNumberID",
		"customer_id":                       a.CustomerID,
		"default_outgoing_source_number_id": defaultOutgoingSourceNumberID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	// validate the number exists and belongs to the agent's customer
	num, err := h.numberGet(ctx, defaultOutgoingSourceNumberID)
	if err != nil {
		log.Errorf("Could not validate the number info. err: %v", err)
		return nil, err
	}
	log.WithField("number", num).Debugf("Retrieved number info. number_id: %s", num.ID)
	if num.CustomerID != a.CustomerID {
		log.Infof("The number does not belong to this customer. number_customer_id: %s", num.CustomerID)
		return nil, fmt.Errorf("%w: the number does not belong to this customer", serviceerrors.ErrPermissionDenied)
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateDefaultOutgoingSourceNumberID(ctx, a.CustomerID, defaultOutgoingSourceNumberID)
	if err != nil {
		log.Errorf("Could not update the customer's default outgoing source number id. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerSelfUpdateMetadata updates the authenticated agent's own customer's metadata.
// Requires CustomerAdmin permission.
func (h *serviceHandler) CustomerSelfUpdateMetadata(ctx context.Context, a *auth.AuthIdentity, metadata cscustomer.Metadata) (*cscustomer.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CustomerSelfUpdateMetadata",
		"customer_id": a.CustomerID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.CustomerV1CustomerUpdateMetadata(ctx, a.CustomerID, metadata)
	if err != nil {
		log.Errorf("Could not update the customer's metadata. err: %v", err)
		return nil, err
	}
	log.WithField("customer", res).Debugf("Updated customer metadata. customer_id: %s", res.ID)

	return res.ConvertWebhookMessage(), nil
}

// CustomerSignup creates an unverified customer and sends a verification email.
// This is a public endpoint — no authentication required.
func (h *serviceHandler) CustomerSignup(
	ctx context.Context,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
	clientIP string,
) (*cscustomer.SignupResultWebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":  "CustomerSignup",
		"email": email,
	})
	log.Debug("Processing customer signup.")

	res, err := h.reqHandler.CustomerV1CustomerSignup(ctx, name, detail, email, phoneNumber, address, webhookMethod, webhookURI, clientIP)
	if err != nil {
		log.Errorf("Could not signup customer. err: %v", err)
		return nil, err
	}

	// Auto-create empty OutboundConfig for the new customer.
	// Empty whitelist blocks all PSTN calls until the customer explicitly configures it.
	// Fire-and-forget: OutboundConfig failure does not block customer signup.
	if res.Customer != nil {
		if _, cfgErr := h.reqHandler.CallV1OutboundConfigCreate(ctx, res.Customer.ID, &cmoutboundconfig.UpdateRequest{}); cfgErr != nil {
			log.Warnf("Could not auto-create OutboundConfig for new customer. customer_id: %s, err: %v", res.Customer.ID, cfgErr)
		}
	}

	return res.ConvertWebhookMessage(), nil
}

// CustomerEmailVerify validates a verification token and activates the customer.
// This is a public endpoint — no authentication required.
func (h *serviceHandler) CustomerEmailVerify(ctx context.Context, token string) (*cscustomer.EmailVerifyResultWebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "CustomerEmailVerify",
	})
	log.Debug("Processing customer email verification.")

	res, err := h.reqHandler.CustomerV1CustomerEmailVerify(ctx, token)
	if err != nil {
		log.Errorf("Could not verify customer email. err: %v", err)
		return nil, err
	}

	return res.ConvertWebhookMessage(), nil
}

// convertCustomerFilters converts map[string]string to map[cscustomer.Field]any
func (h *serviceHandler) convertCustomerFilters(filters map[string]string) (map[cscustomer.Field]any, error) {
	// Convert to map[string]any first
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	// Use reflection-based converter
	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, cscustomer.Customer{})
	if err != nil {
		return nil, err
	}

	// Convert string keys to Field type
	result := make(map[cscustomer.Field]any, len(typed))
	for k, v := range typed {
		result[cscustomer.Field(k)] = v
	}

	return result, nil
}
