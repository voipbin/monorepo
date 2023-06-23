package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
)

// billingAccountGet validates the billing account's ownership and returns the billing account info.
func (h *serviceHandler) billingAccountGet(ctx context.Context, u *cscustomer.Customer, accountID uuid.UUID) (*bmaccount.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "billingAccountGet",
		"customer_id": u.ID,
		"account_id":  accountID,
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

	if !u.HasPermission(cspermission.PermissionAdmin.ID) && u.ID != res.CustomerID {
		log.Info("The user has no permission.")
		return nil, fmt.Errorf("user has no permission")
	}

	return res, nil
}

// BillingAccountGet sends a request to billing-manager
// to getting a billing account.
// it returns billing account if it succeed.
func (h *serviceHandler) BillingAccountGet(ctx context.Context, u *cscustomer.Customer, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountGet",
		"customer_id":        u.ID,
		"username":           u.Username,
		"billing_account_id": billingAccountID,
	})

	// get billing account
	b, err := h.billingAccountGet(ctx, u, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	// convert
	res := b.ConvertWebhookMessage()

	return res, nil
}

// BillingAccountDelete sends a request to billing-manager
// to deleting a billing account.
// it returns billing account if it succeed.
func (h *serviceHandler) BillingAccountDelete(ctx context.Context, u *cscustomer.Customer, billingAccountID uuid.UUID) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountDelete",
		"customer_id":        u.ID,
		"username":           u.Username,
		"billing_account_id": billingAccountID,
	})

	// get billing account
	_, err := h.billingAccountGet(ctx, u, billingAccountID)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
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
func (h *serviceHandler) BillingAccountGets(ctx context.Context, u *cscustomer.Customer, size uint64, token string) ([]*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountGets",
		"customer_id": u.ID,
		"username":    u.Username,
		"size":        size,
		"token":       token,
	})

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	// get billing accounts
	tmps, err := h.reqHandler.BillingV1AccountGets(ctx, u.ID, token, size)
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
func (h *serviceHandler) BillingAccountCreate(ctx context.Context, u *cscustomer.Customer, name string, detail string) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "BillingAccountCreate",
		"customer_id": u.ID,
		"username":    u.Username,
	})

	// get billing accounts
	tmp, err := h.reqHandler.BillingV1AccountCreate(ctx, u.ID, name, detail)
	if err != nil {
		log.Infof("Could not get billing account info. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

// BillingAccountAddBalanceForce sends a request to billing-manager
// to add the given billing account's balance.
func (h *serviceHandler) BillingAccountAddBalanceForce(ctx context.Context, u *cscustomer.Customer, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountAddBalanceForce",
		"customer_id":        u.ID,
		"username":           u.Username,
		"billing_account_id": billingAccountID,
		"balance":            balance,
	})

	// need a admin permission
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission.")
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
func (h *serviceHandler) BillingAccountSubtractBalanceForce(ctx context.Context, u *cscustomer.Customer, billingAccountID uuid.UUID, balance float32) (*bmaccount.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":               "BillingAccountSubtractBalanceForce",
		"customer_id":        u.ID,
		"username":           u.Username,
		"billing_account_id": billingAccountID,
		"balance":            balance,
	})

	// need a admin permission
	if !u.HasPermission(cspermission.PermissionAdmin.ID) {
		log.Info("The user has no permission.")
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
