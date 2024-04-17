package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cmcustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerDeleted",
		"event": m,
	})

	c := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.tagHandler.EventCustomerDeleted(ctx, c); errEvent != nil {
		log.Errorf("Could not handle the groupcall created event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupcall created event.")
	}

	return nil
}
