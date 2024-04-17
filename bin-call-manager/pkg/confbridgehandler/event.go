package confbridgehandler

import (
	"context"

	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *confbridgeHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all calls of the customer. customer_id: %s", cu.ID)

	// get all calls of the customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	confbridges, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets calls list. err: %v", err)
		return errors.Wrap(err, "could not get calls list")
	}

	// terminate all confbridges
	for _, cf := range confbridges {
		log.Debugf("Deleting confbridge info. confbridge_id: %s", cf.ID)
		tmp, err := h.Delete(ctx, cf.ID)
		if err != nil {
			log.Errorf("Could not delete the confbridge. err: %v", err)
			continue
		}
		log.WithField("confbridge", tmp).Debugf("Deleted confbridge. confbridge_id: %s", cf.ID)
	}

	return nil
}
