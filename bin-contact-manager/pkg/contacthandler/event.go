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

	log.Debug("Publishing the contact event.")

	// Publish the event.
	//
	// VOIP-1405 §3.1: the data must be the *WebhookMessage itself, NOT the []byte from
	// CreateWebhookEvent(). PublishEvent marshals whatever it is given, so handing it a []byte
	// double-encoded the payload as a base64 JSON string and, on the global topic exchange, left
	// no top-level `id` to resolve the subscription address from -- every contact event would
	// have published under the `-` placeholder. The stored event history switches from a base64
	// string to a JSON object mid-stream; that is the intended improvement (the sole consumer,
	// bin-timeline-manager, stores the payload verbatim and contact is not in its peer_event
	// whitelist, so nothing breaks).
	h.notifyHandler.PublishEvent(ctx, eventType, c.ConvertWebhookMessage())
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
