package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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

	if errEvent := h.agentHandler.EventCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event.")
	}

	return nil
}

// processEventCMCustomerCreated handles the customer-manager's customer_created event.
func (h *subscribeHandler) processEventCMCustomerCreated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerCreated",
		"event": m,
	})

	// Parse with headless field
	var event struct {
		cmcustomer.Customer
		Headless bool `json:"headless"`
	}
	if err := json.Unmarshal([]byte(m.Data), &event); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	cu := &event.Customer
	if errEvent := h.agentHandler.EventCustomerCreated(ctx, cu, event.Headless); errEvent != nil {
		log.Errorf("Could not handle the customer created event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer created event.")
	}

	return nil
}
