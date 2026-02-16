package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventCMCustomerDeleted handles the customer-manager's customer_deleted event
func (h *subscribeHandler) processEventCMCustomerDeleted(ctx context.Context, m *sock.Event) error {
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

// processEventCMCustomerCreated handles the customer-manager's customer_created event
func (h *subscribeHandler) processEventCMCustomerCreated(ctx context.Context, m *sock.Event) error {
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

	if errEvent := h.accountHandler.EventCUCustomerCreated(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the subscribed event. err: %v", errEvent)
		return errEvent
	}

	return nil
}


// processEventCUCustomerFrozen handles the customer-manager's customer_frozen event
func (h *subscribeHandler) processEventCUCustomerFrozen(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCUCustomerFrozen",
		"event": m,
	})
	log.Debugf("Received customer event. event: %s", m.Type)

	var c cscustomer.Customer
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.accountHandler.EventCUCustomerFrozen(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the subscribed event. err: %v", errEvent)
		return errEvent
	}

	return nil
}

// processEventCUCustomerRecovered handles the customer-manager's customer_recovered event
func (h *subscribeHandler) processEventCUCustomerRecovered(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCUCustomerRecovered",
		"event": m,
	})
	log.Debugf("Received customer event. event: %s", m.Type)

	var c cscustomer.Customer
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.accountHandler.EventCUCustomerRecovered(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the subscribed event. err: %v", errEvent)
		return errEvent
	}

	return nil
}
