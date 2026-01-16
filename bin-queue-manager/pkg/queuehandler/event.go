package queuehandler

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-queue-manager/models/queue"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *queueHandler) EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": cu,
	})
	log.Debugf("Deleting all queues in customer. customer_id: %s", cu.ID)

	// get all queues in customer
	filters := map[queue.Field]any{
		queue.FieldCustomerID: cu.ID,
		queue.FieldDeleted:    false,
	}
	qs, err := h.List(ctx, 1000, "", filters)
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
