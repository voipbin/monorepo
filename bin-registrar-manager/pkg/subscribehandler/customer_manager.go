package subscribehandler

import (
	"context"
	"encoding/json"

	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
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

	return nil
}
