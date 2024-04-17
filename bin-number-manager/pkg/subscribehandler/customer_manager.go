package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCUCustomerDeleted handles the customer-manager's customer_deleted event.
func (h *subscribeHandler) processEventCUCustomerDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCUCustomerDeleted",
		"event": m,
	})

	c := &cmcustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	log.WithField("customer", c).Debugf("Received customer deleted event. customer_id: %s", c.ID)
	if errRemove := h.numberHandler.EventCustomerDeleted(ctx, c); errRemove != nil {
		log.Errorf("Could not handle the customer deleted event. err: %v", errRemove)
		return errors.Wrap(errRemove, "Could not handle the customer deleted event.")
	}

	return nil
}
