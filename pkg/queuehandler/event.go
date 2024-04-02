package queuehandler

import (
	"context"

	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *queueHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all queues in customer. customer_id: %s", cu.ID)

	// get all queues in customer
	filters := map[string]string{
		"customer_id": cu.ID.String(),
		"deleted":     "false",
	}
	qs, err := h.Gets(ctx, 1000, "", filters)
	if err != nil {
		log.Errorf("Could not gets queues list. err: %v", err)
		return errors.Wrap(err, "could not get queues list")
	}

	// delete all queues
	for _, q := range qs {
		log.Debugf("Deleting queue info. queue_id: %s", q.ID)
		tmp, err := h.Delete(ctx, q.ID)
		if err != nil {
			log.Errorf("Could not delete queue info. err: %v", err)
			continue
		}
		log.WithField("queue", tmp).Debugf("Deleted queue info. queue_id: %s", tmp.ID)
	}

	return nil
}
