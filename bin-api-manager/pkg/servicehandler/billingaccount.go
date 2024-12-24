package servicehandler

import (
	"context"
	"fmt"

	bmaccount "monorepo/bin-billing-manager/models/account"

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

	if res.TMDelete < defaultTimestamp {
		log.Debugf("Deleted billing_account. billing_account_id: %s", res.ID)
		return nil, fmt.Errorf("not found")
	}

	return res, nil
}

// BillingAccountGet sends a request to billing-manager
// to getting a billing account.
// it returns billing account if it succeed.
func (h *serviceHandler) BillingAccountGet(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountGet",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
	})

	// get billing account
	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// convert
	res := ba.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountDelete sends a request to billing-manager
// to deleting a billing account.
// it returns billing account if it succeed.
func (h *serviceHandler) BillingAccountDelete(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountDelete",
		"customer_id":        a.CustomerID,
		"username":           a.Username,
		"billing_account_id": billingAccountID,
	})

	// get billing account
	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	tmp, err := h.reqHandler.BillingV1AccountDelete(ctx, billingAccountID)
	if err != nil {
		log.Errorf("Could not delete billing account. err: %v", err)
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountGets sends a request to billing-manager
// to getting a list of billing accounts.
// it returns list of billing accounts if it succeed.
func (h *serviceHandler) BillingAccountGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountGets",
		"customer_id": a.CustomerID,
		"username":    a.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// filters
	filters := map[string]string{
		"customer_id": a.CustomerID.String(),
		"deleted":     "false", // we don't need deleted items
	}

	// get billing accounts
	tmps, err := h.reqHandler.BillingV1AccountGets(ctx, token, size, filters)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	// create result
	res := []*bmaccount.WebhookMessage{}
	for _, tmp := range tmps {
		c := tmp.ConvertWebhookMessage()
		res = append(res, c)
	}

	return res, nil
}

// BillingAccountCreate sends a request to billing-manager
// to create a new billing accounts.
// it returns created billing account if it succeed.
func (h *serviceHandler) BillingAccountCreate(ctx context.Context, a *amagent.Agent, name string, detail string, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountCreate",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get billing accounts
	tmp, err := h.reqHandler.BillingV1AccountCreate(ctx, a.CustomerID, name, detail, paymentType, paymentMethod)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountUpdateBasicInfo sends a request to billing-manager
// to update the billing account's basic info.
// it returns updated billing account if it succeed.
func (h *serviceHandler) BillingAccountUpdateBasicInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, name string, detail string) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountUpdateBasicInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	// get billing account
	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get billing accounts
	tmp, err := h.reqHandler.BillingV1AccountUpdateBasicInfo(ctx, billingAccountID, name, detail)
	if err != nil {
		log.Infof("Could not get update account info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountUpdateBasicInfo sends a request to billing-manager
// to update the billing account's basic info.
// it returns updated billing account if it succeed.
func (h *serviceHandler) BillingAccountUpdatePaymentInfo(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountUpdatePaymentInfo",
		"customer_id": a.CustomerID,
		"username":    a.Username,
	})

	// get billing account
	ba, err := h.billingAccountGet(ctx, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ba.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// get billing accounts
	tmp, err := h.reqHandler.BillingV1AccountUpdatePaymentInfo(ctx, billingAccountID, paymentType, paymentMethod)
	if err != nil {
		log.Infof("Could not get update account info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountAddBalanceForce sends a request to billing-manager
// to add the given billing account's balance.
func (h *serviceHandler) BillingAccountAddBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error) {
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

	res := b.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountSubtractBalanceForce sends a request to billing-manager
// to subtract the given billing account's balance.
func (h *serviceHandler) BillingAccountSubtractBalanceForce(ctx context.Context, a *amagent.Agent, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error) {
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

	res := b.ConvertWebhookMessage()
	return res, nil
}
