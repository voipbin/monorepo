package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// processEventCMCustomerCreated handles the customer-manager's customer_created event
func (h *subscribeHandler) processEventCMCustomerCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCustomerCreated",
		"event": m,
	})
	log.Debugf("Received customer event. event: %s", m.Type)

	var c cscustomer.Customer
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	a, err := h.accountHandler.Create(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not craete a new account. err: %v", err)
		return errors.Wrap(err, "could not create a new aacount")
	}
	log.WithField("account", a).Debugf("Created a new account. account_id: %s", a.ID)

	return nil
}
