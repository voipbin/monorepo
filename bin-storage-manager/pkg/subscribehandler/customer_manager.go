package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMCustomerCreated handles the customer-manager's customer_created event.
func (h *subscribeHandler) processEventCMCustomerCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerCreated",
		"event": m,
	})

	cu := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.accountHandler.EventCustomerCreated(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer created event by the account handler. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer created event by the account handler.")
	}

	return nil
}

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerDeleted",
		"event": m,
	})

	cu := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.fileHandler.EventCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event by the file handler. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event by the file handler.")
	}

	if errEvent := h.accountHandler.EventCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event by the account handler. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event by the account handler.")
	}

	return nil
}
