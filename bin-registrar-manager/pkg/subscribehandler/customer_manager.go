package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMCustomerCreated handles the customer-manager's customer_created event.
// Provisions the customer's SIP domain row (idempotent).
func (h *subscribeHandler) processEventCMCustomerCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerCreated",
		"event": m,
	})

	cu := &cucustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.customerDomainHandler.EventCUCustomerCreated(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer created event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer created event.")
	}

	return nil
}

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerDeleted",
		"event": m,
	})

	cu := &cucustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.trunkHandler.EventCUCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event.")
	}

	if errEvent := h.extensionHandler.EventCUCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event.")
	}

	// delete the customer's SIP domain row after the trunk/extension cascade
	if errEvent := h.customerDomainHandler.EventCUCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event.")
	}

	return nil
}
