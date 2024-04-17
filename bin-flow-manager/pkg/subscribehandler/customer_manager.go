package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerDeleted",
		"event": m,
	})

	cu := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.flowHandler.EventCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event by the flow handler. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event by the flow handler.")
	}

	if errEvent := h.activeflowHandler.EventCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event. by activeflow handler. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event by the activeflow handler")
	}

	return nil
}
