package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
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

	accounts, err := h.accountHandler.DeletesByCustomerID(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not craete a new account. err: %v", err)
		return errors.Wrap(err, "could not create a new aacount")
	}
	log.WithField("accounts", accounts).WithField("account", accounts).Debugf("Deleted customer accounts. customer_id: %s", c.ID)

	return nil
}
