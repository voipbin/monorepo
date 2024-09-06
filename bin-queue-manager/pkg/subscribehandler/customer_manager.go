package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCUCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCUCustomerDeleted(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCUCustomerDeleted",
		"event": m,
	})

	cu := &cucustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &cu); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	// queuecall handler
	if errEvent := h.queuecallHandler.EventCUCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event by the queuecall handler. err: %v", errEvent)
		return errors.Wrap(errEvent, "could not handle the customer deleted event by the queuecall handler")
	}

	// queue handler
	if errEvent := h.queueHandler.EventCUCustomerDeleted(ctx, cu); errEvent != nil {
		log.Errorf("Could not handle the customer deleted event by the queuehandler. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the customer deleted event by the queuehandler.")
	}

	return nil
}
