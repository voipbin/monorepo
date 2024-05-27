package accounthandler

import (
	"context"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCustomerCreated handles the customer-manager's customer_created event
func (h *accountHandler) EventCustomerCreated(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerCreated",
		"customer": cu,
	})
	log.Debugf("Creating a new account for customer. customer_id: %s", cu.ID)

	tmp, err := h.Create(ctx, cu.ID)
	if err != nil {
		log.Errorf("Could not create account for a new customer. err: %v", err)
		return errors.Wrap(err, "Could not create account for a new customer")
	}
	log.WithField("account", tmp).Debugf("Created an account for a new customer. account_id: %s", tmp.ID)

	return nil
}

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *accountHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting customer's all files. customer_id: %s", cu.ID)

	// get account
	a, err := h.getByCustomerID(ctx, cu.ID)
	if err != nil {
		log.Errorf("Could not find account info for customer. err: %v", err)
		return errors.Wrap(err, "could not find account info for customer")
	}

	// delete account
	tmp, err := h.Delete(ctx, a.ID)
	if err != nil {
		log.Errorf("Could not delete the account info. err: %v", err)
		return err
	}
	log.WithField("account", tmp).Debugf("Deleted account info. account_id: %s", tmp.ID)

	return nil
}
