package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// processEventCSCustomerCreatedUpdated handles the customer-manager's customer_created and customer_updated event.
func (h *subscribeHandler) processEventCSCustomerCreatedUpdated(ctx context.Context, m *rabbitmqhandler.Event) error {
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
		log.Errorf("Could not update the messagetarget. err: %v", err)
		return err
	}
	log.WithField("account", tmp).Debugf("Updated account. account_id: %s", tmp.ID)

	return nil
}
