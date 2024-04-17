package groupcallhandler

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCUCustomerDeleted handles the customer-manager's customer_deleted event
func (h *groupcallHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCUCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all groupcalls of the customer. customer_id: %s", cu.ID)

	// get all groupcalls of the customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	groupcalls, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets calls list. err: %v", err)
		return errors.Wrap(err, "could not get calls list")
	}

	// delete all groupcalls
	for _, e := range groupcalls {
		log.Debugf("Deleting groupcall info. groupcall_id: %s", e.ID)
		tmp, err := h.Delete(ctx, e.ID)
		if err != nil {
			log.Errorf("Could not delete groupcall info. err: %v", err)
			continue
		}
		log.WithField("groupcall", tmp).Debugf("Deleted groupcall info. groupcall_id: %s", tmp.ID)
	}

	return nil
}
