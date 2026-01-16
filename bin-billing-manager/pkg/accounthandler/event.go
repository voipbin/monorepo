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

	return nil
}
