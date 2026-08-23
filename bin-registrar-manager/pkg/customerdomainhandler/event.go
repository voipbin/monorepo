package customerdomainhandler

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerCreated handles the customer-manager's customer_created event:
// provisions the customer's domain row so the mapping exists before the first
// extension is created. Idempotent (EnsureByCustomerID).
func (h *customerDomainHandler) EventCUCustomerCreated(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCUCustomerCreated",
		"customer_id": cu.ID,
	})
	log.Debugf("Ensuring the customer domain of the customer. customer_id: %s", cu.ID)

	res, err := h.EnsureByCustomerID(ctx, cu.ID)
	if err != nil {
		log.Errorf("Could not ensure the customer domain. err: %v", err)
		return errors.Wrap(err, "could not ensure the customer domain")
	}
	log.Debugf("Ensured customer domain. label: %s, realm: %s", res.DomainLabel, res.Realm)

	return nil
}

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event:
// hard-deletes the customer's domain row (runs after the trunk/extension cascade).
func (h *customerDomainHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCUCustomerDeleted",
		"customer_id": cu.ID,
	})
	log.Debugf("Deleting the customer domain of the customer. customer_id: %s", cu.ID)

	if err := h.DeleteByCustomerID(ctx, cu.ID); err != nil {
		log.Errorf("Could not delete the customer domain. err: %v", err)
		return errors.Wrap(err, "could not delete the customer domain")
	}

	return nil
}
