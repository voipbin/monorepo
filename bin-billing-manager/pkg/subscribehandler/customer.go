package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerDeleted",
		"event": m,
	})
	log.Debugf("Received customer event. event: %s", m.Type)

	var c cscustomer.Customer
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.accountHandler.EventCUCustomerDeleted(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the subscribed event. err: %v", errEvent)
		return errEvent
	}

	return nil
}
