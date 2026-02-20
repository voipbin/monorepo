package accounthandler

import (
	"context"

	"monorepo/bin-billing-manager/models/account"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *accountHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all accounts of the customer. customer_id: %s", cu.ID)

	// get all accounts of the customer
	filters := map[account.Field]any{
		account.FieldCustomerID: cu.ID,
		account.FieldDeleted:    false,
	}
	accounts, err := h.List(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets accounts list. err: %v", err)
		return errors.Wrap(err, "could not get accounts list")
	}

	// delete all accounts
	for _, a := range accounts {
		log.Debugf("Deleting account info. account_id: %s", a.ID)
		tmp, err := h.Delete(ctx, a.ID)
		if err != nil {
			log.Errorf("Could not delete account info. err: %v", err)
			continue
		}
		log.WithField("account", tmp).Debugf("Deleted account info. account_id: %s", tmp.ID)
	}

	return nil
}

// EventCUCustomerCreated handles the customer-manager's customer_created event
func (h *accountHandler) EventCUCustomerCreated(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerCreated",
		"customer": cu,
	})
	log.Debugf("Creatinga a new billing account for new customer. customer_id: %s", cu.ID)

	b, err := h.Create(ctx, cu.ID,
		"basic billing account",
		"billing account for default use",
		account.PaymentTypePrepaid,
		account.PaymentMethodNone,
	)
	if err != nil {
		log.Errorf("Could not create a basic billing account. err: %v", err)
		return errors.Wrap(err, "could not create a basic billing account")
	}
	log.WithField("billing_account", b).Debugf("Created a basic billing account. account_id: %s", b.ID)

	// set default billing account for customer
	tmp, err := h.reqHandler.CustomerV1CustomerUpdateBillingAccountID(ctx, cu.ID, b.ID)
	if err != nil {
		log.Errorf("Could not update customer's billing account id. err: %v", err)
		return errors.Wrap(err, "could not update customer's billing account id")
	}
	log.WithField("customer", tmp).Debugf("Updated customer's billing account id. customer_id: %s, billing_account_id: %s", tmp.ID, tmp.BillingAccountID)

	// set default plan type for new account
	if _, errPlan := h.dbUpdatePlanType(ctx, b.ID, account.PlanTypeFree); errPlan != nil {
		log.Errorf("Could not set default plan type. err: %v", errPlan)
		// non-fatal: account is created, customer can still use the platform
	}

	// initial token topup for new customer
	tokenAmount, ok := account.PlanTokenMap[account.PlanTypeFree]
	if ok && tokenAmount > 0 {
		if errTopup := h.db.AccountTopUpTokens(ctx, b.ID, cu.ID, tokenAmount, string(account.PlanTypeFree)); errTopup != nil {
			log.Errorf("Could not perform initial token topup. err: %v", errTopup)
			// non-fatal: account is created, tokens can be topped up later
		}
	}

	return nil
}

// EventCUCustomerFrozen handles the customer-manager's customer_frozen event
func (h *accountHandler) EventCUCustomerFrozen(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerFrozen",
		"customer": cu,
	})
	log.Debugf("Freezing all accounts of the customer. customer_id: %s", cu.ID)

	// get all active accounts for the customer
	filters := map[account.Field]any{
		account.FieldCustomerID: cu.ID,
		account.FieldDeleted:    false,
	}
	accounts, err := h.List(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get accounts list. err: %v", err)
		return errors.Wrap(err, "could not get accounts list")
	}

	// set status='frozen' for each account (does NOT set tm_delete)
	for _, a := range accounts {
		log.Debugf("Freezing account. account_id: %s", a.ID)
		if errSet := h.db.AccountSetStatus(ctx, a.ID, account.StatusFrozen); errSet != nil {
			log.Errorf("Could not set account status. account_id: %s, err: %v", a.ID, errSet)
		}
	}

	return nil
}

// EventCUCustomerRecovered handles the customer-manager's customer_recovered event
func (h *accountHandler) EventCUCustomerRecovered(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerRecovered",
		"customer": cu,
	})
	log.Debugf("Recovering frozen accounts of the customer. customer_id: %s", cu.ID)

	// get all frozen accounts for the customer (not all accounts - only frozen ones)
	filters := map[account.Field]any{
		account.FieldCustomerID: cu.ID,
		account.FieldStatus:     account.StatusFrozen,
	}
	accounts, err := h.List(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not get accounts list. err: %v", err)
		return errors.Wrap(err, "could not get accounts list")
	}

	// set status='active' for each frozen account
	for _, a := range accounts {
		log.Debugf("Recovering account. account_id: %s", a.ID)
		if errSet := h.db.AccountSetStatus(ctx, a.ID, account.StatusActive); errSet != nil {
			log.Errorf("Could not set account status. account_id: %s, err: %v", a.ID, errSet)
		}
	}

	return nil
}
