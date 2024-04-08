package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallProgressing handles the call-manager's call_created event
func (h *subscribeHandler) processEventCMCallProgressing(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallProgressing",
		"event": m,
	})
	log.Debugf("Received call event. event: %s", m.Type)

	var c cmcall.Call
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventCMCallProgressing(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	// if errBilling := h.billingHandler.BillingStart(ctx, c.CustomerID, billing.ReferenceTypeCall, c.ID, c.TMProgressing, &c.Source, &c.Destination); errBilling != nil {
	// 	log.Errorf("Could not create a billing. err: %v", errBilling)
	// 	return errors.Wrap(errBilling, "could not create a billing")
	// }

	return nil
}

// processEventCMCallHangup handles the call-manager's call_hangup event
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallHangup",
		"event": m,
	})
	log.Debugf("Received call event. event: %s", m.Type)

	var c cmcall.Call
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return errors.Wrap(err, "could not unmarshal the data")
	}

	if errEvent := h.billingHandler.EventCMCallHangup(ctx, &c); errEvent != nil {
		log.Errorf("Could not handle the event. err: %v", errEvent)
		return errEvent
	}

	// _, err := h.billingHandler.GetByReferenceID(ctx, c.ID)
	// if err != nil {
	// 	// could not get billing. nothing to do.
	// 	return nil
	// }

	// if errBilling := h.billingHandler.BillingEndByReferenceID(ctx, c.ID, c.TMHangup, &c.Source, &c.Destination); errBilling != nil {
	// 	log.Errorf("Could not end the billing. err: %v", errBilling)
	// 	return errors.Wrap(errBilling, "could not end the billing")
	// }

	return nil
}
