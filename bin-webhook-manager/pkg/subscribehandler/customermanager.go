package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

// processEventCSCustomerCreatedUpdated handles the customer-manager's customer_created and customer_updated event.
func (h *subscribeHandler) processEventCSCustomerCreatedUpdated(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventCSCustomerCreatedUpdated",
			"event": m,
		},
	)
	log.Debugf("Received customer event. event: %s", m.Type)

	e := &cscustomer.Customer{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	tmp, err := h.accountHandler.UpdateByCustomer(ctx, e)
	if err != nil {
		log.Errorf("Could not update the account. err: %v", err)
		return err
	}
	log.Debugf("Updated account. account: %v", tmp)

	return nil
}
