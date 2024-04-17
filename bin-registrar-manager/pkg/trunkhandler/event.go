package trunkhandler

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *trunkHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all trunks of the customer. customer_id: %s", cu.ID)

	// get all trunks in customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	trunks, err := h.Gets(ctx, "", 1000, filters)
	if err != nil {
		log.Errorf("Could not gets trunks list. err: %v", err)
		return errors.Wrap(err, "could not get trunks list")
	}

	// delete all trunks
	for _, t := range trunks {
		log.Debugf("Deleting trunk info. trunk_id: %s", t.ID)
		tmp, err := h.Delete(ctx, t.ID)
		if err != nil {
			log.Errorf("Could not delete trunk info. err: %v", err)
			continue
		}
		log.WithField("trunk", tmp).Debugf("Deleted trunk info. trunk_id: %s", tmp.ID)
	}

	return nil
}
