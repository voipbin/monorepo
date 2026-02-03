package contacthandler

import (
	"context"

	"github.com/sirupsen/logrus"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-contact-manager/models/contact"
)

// publishEvent publishes a contact event
func (h *contactHandler) publishEvent(ctx context.Context, eventType string, c *contact.Contact) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "publishEvent",
		"event_type": eventType,
		"contact_id": c.ID,
	})

	// Create webhook event data
	data, err := c.CreateWebhookEvent()
	if err != nil {
		log.Errorf("Could not create webhook event. err: %v", err)
		return
	}

	// Publish the event
	h.notifyHandler.PublishEvent(ctx, eventType, data)
}

// EventCustomerDeleted handles customer deletion by cleaning up all contacts
func (h *contactHandler) EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "EventCustomerDeleted",
		"customer_id": c.ID,
	})

	log.Info("Customer deleted. Cleaning up contacts.")

	// Delete all contacts for this customer
	if err := h.db.ContactDeleteByCustomerID(ctx, c.ID); err != nil {
		log.Errorf("Could not delete contacts for customer. err: %v", err)
		return err
	}

	log.Info("Successfully cleaned up contacts for deleted customer.")
	return nil
}
