package taghandler

import (
	"context"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"

	"monorepo/bin-tag-manager/models/tag"
)

// EventCustomerDeleted handles the customer-manager's customer_deleted event
func (h *tagHandler) EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":     "EventCustomerDeleted",
		"customer": c,
	})

	// build filters for the customer's tags
	filters := map[tag.Field]any{
		tag.FieldCustomerID: c.ID,
		tag.FieldDeleted:    false,
	}

	// get all tags of the customer
	tags, err := h.dbGets(ctx, 999, h.utilHandler.TimeGetCurTime(), filters)
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
