package accounthandler

import (
	"context"

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
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	accounts, err := h.Gets(ctx, 1000, "", filters)
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
