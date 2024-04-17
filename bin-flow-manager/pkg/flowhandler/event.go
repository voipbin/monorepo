package flowhandler

import (
	"context"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *flowHandler) EventCustomerDeleted(ctx context.Context, cu *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all flows of customer. customer_id: %s", cu.ID)

	// get all flows in customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	ags, err := h.Gets(ctx, h.util.TimeGetCurTime(), 1000, filters)
	if err != nil {
		log.Errorf("Could not gets flows list. err: %v", err)
		return errors.Wrap(err, "could not get flows list")
	}

	// delete all flows
	for _, f := range ags {
		log.Debugf("Deleting flow info. flow_id: %s", f.ID)
		tmp, err := h.Delete(ctx, f.ID)
		if err != nil {
			log.Errorf("Could not delete flow info. err: %v", err)
			continue
		}
		log.WithField("flow", tmp).Debugf("Deleted flow info. flow_id: %s", tmp.ID)
	}

	return nil
}
