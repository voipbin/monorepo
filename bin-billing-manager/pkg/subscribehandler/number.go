package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// processEventNMNumberCreated handles the number-manager's number_created event
func (h *subscribeHandler) processEventNMNumberCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventNMNumberCreated",
		"event": m,
	})
	log.Debugf("Received call event. event: %s", m.Type)

	var c nmnumber.Number
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventNMNumberCreated(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	// if errBilling := h.billingHandler.BillingStart(ctx, c.CustomerID, billing.ReferenceTypeNumber, c.ID, c.TMCreate, &commonaddress.Address{}, &commonaddress.Address{}); errBilling != nil {
	// 	log.Errorf("Could not create a billing. number_id: %s", c.ID)
	// 	return errors.Wrap(errBilling, "could not create a billing")
	// }

	return nil
}

// processEventNMNumberRenewed handles the number-manager's number_renewed event
func (h *subscribeHandler) processEventNMNumberRenewed(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventNMNumberRenewed",
		"event": m,
	})
	log.Debugf("Received call event. event: %s", m.Type)

	var c nmnumber.Number
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventNMNumberRenewed(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	// if errBilling := h.billingHandler.BillingStart(ctx, c.CustomerID, billing.ReferenceTypeNumber, c.ID, c.TMCreate, &address.Address{}, &address.Address{}); errBilling != nil {
	// 	log.Errorf("Could not create a billing. number_id: %s", c.ID)
	// 	return errors.Wrap(errBilling, "could not create a billing")
	// }

	return nil
}
