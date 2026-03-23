package servicehandler

import (
	"context"
	"fmt"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// billingAccountGet validates the billing account's ownership and returns the billing account info.
func (h *serviceHandler) billingAccountGet(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "billingAccountGet",
		"account_id": accountID,
	})

	// send request
	res, err := h.reqHandler.BillingV1AccountGet(ctx, accountID)
	if err != nil {
		log.Errorf("Could not get the billing account info. err: %v", err)
		return nil, err
	}
	log.WithField("billing_account", res).Debug("Received result.")

	if res.TMDelete != nil {
		log.Debugf("Deleted billing_account. billing_account_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// BillingAccountGet sends a request to billing-manager
// to getting a billing account.
// it returns billing account if it succeed.
func (h *serviceHandler) BillingAccountGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountGet",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
	})

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get billing account
	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	return ba, nil
}

// BillingAccountUpdateBasicInfo sends a request to billing-manager
// to update the billing account's basic info.
// it returns updated billing account if it succeed.
func (h *serviceHandler) BillingAccountUpdateBasicInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, name string, detail string) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// update billing account
	tmp, err := h.reqHandler.BillingV1AccountUpdateBasicInfo(ctx, billingAccountID, name, detail)
	if err != nil {
		log.Infof("Could not update account info. err: %v", err)
		return nil, err
	}

	return tmp, nil
}

// BillingAccountUpdatePaymentInfo sends a request to billing-manager
// to update the billing account's payment info.
// it returns updated billing account if it succeed.
func (h *serviceHandler) BillingAccountUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountUpdatePaymentInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// update billing account payment info
	tmp, err := h.reqHandler.BillingV1AccountUpdatePaymentInfo(ctx, billingAccountID, paymentType, paymentMethod)
	if err != nil {
		log.Infof("Could not update account payment info. err: %v", err)
		return nil, err
	}

	return tmp, nil
}

// BillingAccountAddBalanceForce sends a request to billing-manager
// to add the given billing account's balance.
func (h *serviceHandler) BillingAccountAddBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance int64) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountAddBalanceForce",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
		"balance":            balance,
	})

	// need a project super admin permission
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	b, err := h.reqHandler.BillingV1AccountAddBalanceForce(ctx, billingAccountID, balance)
	if err != nil {
		log.Errorf("Could not add the balance. err: %v", err)
		return nil, errors.Wrap(err, "could not add the balance")
	}

	return b, nil
}

// BillingAccountSubtractBalanceForce sends a request to billing-manager
// to subtract the given billing account's balance.
func (h *serviceHandler) BillingAccountSubtractBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance int64) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountSubtractBalanceForce",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
		"balance":            balance,
	})

	// need a project super admin permission
	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	b, err := h.reqHandler.BillingV1AccountSubtractBalanceForce(ctx, billingAccountID, balance)
	if err != nil {
		log.Errorf("Could not subtract the balance. err: %v", err)
		return nil, errors.Wrap(err, "could not subtract the balance")
	}

	return b, nil
}

// BillingAccountSelfGet returns the authenticated agent's own billing account.
func (h *serviceHandler) BillingAccountSelfGet(ctx context.Context, a *amagent.Agent) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountSelfGet",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	// get customer to resolve billing account ID
	c, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.BillingAccountID == uuid.Nil {
		log.Info("Customer has no billing account.")
		return nil, fmt.Errorf("no billing account")
	}

	ba, err := h.billingAccountGet(ctx, c.BillingAccountID)
	if err != nil {
		log.Errorf("Could not get the billing account info. err: %v", err)
		return nil, err
	}
	log.WithField("billing_account", ba).Debugf("Retrieved billing account info. billing_account_id: %s", ba.ID)

	return ba.ConvertWebhookMessage(), nil
}

// BillingAccountSelfUpdateBasicInfo updates the authenticated agent's own billing account's basic info.
func (h *serviceHandler) BillingAccountSelfUpdateBasicInfo(ctx context.Context, a *amagent.Agent, name string, detail string) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountSelfUpdateBasicInfo",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	c, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.BillingAccountID == uuid.Nil {
		log.Info("Customer has no billing account.")
		return nil, fmt.Errorf("no billing account")
	}

	tmp, err := h.reqHandler.BillingV1AccountUpdateBasicInfo(ctx, c.BillingAccountID, name, detail)
	if err != nil {
		log.Infof("Could not update account info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// BillingAccountSelfUpdatePaymentInfo updates the authenticated agent's own billing account's payment info.
func (h *serviceHandler) BillingAccountSelfUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountSelfUpdatePaymentInfo",
		"customer_id": a.CustomerID,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		log.Info("The agent has no permission.")
		return nil, fmt.Errorf("agent has no permission")
	}

	c, err := h.customerGet(ctx, a.CustomerID)
	if err != nil {
		log.Errorf("Could not get the customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.BillingAccountID == uuid.Nil {
		log.Info("Customer has no billing account.")
		return nil, fmt.Errorf("no billing account")
	}

	tmp, err := h.reqHandler.BillingV1AccountUpdatePaymentInfo(ctx, c.BillingAccountID, paymentType, paymentMethod)
	if err != nil {
		log.Infof("Could not update account payment info. err: %v", err)
		return nil, err
	}

	return tmp.ConvertWebhookMessage(), nil
}

// BillingAccountList returns a list of all billing accounts.
func (h *serviceHandler) BillingAccountList(ctx context.Context, a *amagent.Agent, size uint64, token string, filters map[string]string) ([]*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "BillingAccountList",
		"agent":   a,
		"size":    size,
		"token":   token,
		"filters": filters,
	})
	log.Debug("Received request detail.")

	if !h.hasPermission(ctx, a, uuid.Nil, amagent.PermissionProjectSuperAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if size <= 0 {
		size = 10
	}
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// Convert string filters to typed filters
	typedFilters, err := h.convertBillingAccountFilters(filters)
	if err != nil {
		return nil, err
	}

	tmps, err := h.reqHandler.BillingV1AccountGets(ctx, token, size, typedFilters)
	if err != nil {
		log.Errorf("Could not get billing accounts info. err: %v", err)
		return nil, err
	}

	res := make([]*bmaccount.Account, len(tmps))
	for i := range tmps {
		res[i] = &tmps[i]
	}

	return res, nil
}

// convertBillingAccountFilters converts map[string]string to map[bmaccount.Field]any
func (h *serviceHandler) convertBillingAccountFilters(filters map[string]string) (map[bmaccount.Field]any, error) {
	srcAny := make(map[string]any, len(filters))
	for k, v := range filters {
		srcAny[k] = v
	}

	typed, err := commondatabasehandler.ConvertMapToTypedMap(srcAny, bmaccount.Account{})
	if err != nil {
		return nil, err
	}

	result := make(map[bmaccount.Field]any, len(typed))
	for k, v := range typed {
		result[bmaccount.Field(k)] = v
	}

	return result, nil
}

