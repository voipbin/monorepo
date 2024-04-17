package taghandler

import (
	"context"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *tagHandler) EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": c,
	})

	// get all tags of the customer
	tags, err := h.dbGets(ctx, c.ID, 999, h.utilHandler.TimeGetCurTime())
	if err != nil {
		log.Errorf("")
	}

	for _, t := range tags {
		tmp, err := h.Delete(ctx, t.ID)
		if err != nil {
			log.Errorf("Could not delete the tag: %v", err)
			continue
		}

		log.WithField("tag", tmp).Debugf("Deleted tag. tag_id: %s", tmp.ID)
	}

	return nil
}
