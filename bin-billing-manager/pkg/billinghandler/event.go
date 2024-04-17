package billinghandler

import (
	"context"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	mmmessage "monorepo/bin-message-manager/models/message"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCMCallProgressing handles the call-manager's call_progressing event
func (h *billingHandler) EventCMCallProgressing(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCMCallProgressing",
		"call": c,
	})
	log.Debugf("Received call_progressing event. call_id: %s", c.ID)

	if errBilling := h.BillingStart(ctx, c.CustomerID, billing.ReferenceTypeCall, c.ID, c.TMProgressing, &c.Source, &c.Destination); errBilling != nil {
		log.Errorf("Could not start a billing. err: %v", errBilling)
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *billingHandler) EventCMCallHangup(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "EventCMCallHangup",
		"call": c,
	})
	log.Debugf("Received call_hangup event. call_id: %s", c.ID)

	// get billing info
	b, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if errEnd := h.BillingEnd(ctx, b, c.TMHangup, &c.Source, &c.Destination); errEnd != nil {
		log.Errorf("Could not end the billing. err: %v", errEnd)
		return errors.Wrap(errEnd, "could not end the billing")
	}

	return nil
}

// EventMMMessageCreated handles the message-manager's message_created event
func (h *billingHandler) EventMMMessageCreated(ctx context.Context, m *mmmessage.Message) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "EventMMMessageCreated",
		"message": m,
	})
	log.Debugf("Received message_created event. message_id: %s", m.ID)

	for _, target := range m.Targets {
		log.WithField("target", target).Debugf("Creating billing for message. destination: %v", target.Destination)
		if errBilling := h.BillingStart(ctx, m.CustomerID, billing.ReferenceTypeSMS, m.ID, m.TMCreate, m.Source, &target.Destination); errBilling != nil {
			log.Errorf("Could not create a billing. target: %v, err: %v", target, errBilling)
			return errors.Wrap(errBilling, "could not create a billing")
		}
	}

	return nil
}

// EventNMNumberCreated handles the number-manager's number_created event
func (h *billingHandler) EventNMNumberCreated(ctx context.Context, n *nmnumber.Number) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "EventNMNumberCreated",
		"number": n,
	})
	log.Debugf("Received number_created event. number_id: %s", n.ID)

	if errBilling := h.BillingStart(ctx, n.CustomerID, billing.ReferenceTypeNumber, n.ID, n.TMCreate, &commonaddress.Address{}, &commonaddress.Address{}); errBilling != nil {
		log.Errorf("Could not create a billing. number_id: %s", n.ID)
		return errors.Wrap(errBilling, "could not create a billing")
	}

	return nil
}

// processEventNMNumberRenewed handles the number-manager's number_renewed event
func (h *billingHandler) EventNMNumberRenewed(ctx context.Context, n *nmnumber.Number) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "EventNMNumberRenewed",
		"number": n,
	})
	log.Debugf("Received number_renewed event. number_id: %s", n.ID)

	if errBilling := h.BillingStart(ctx, n.CustomerID, billing.ReferenceTypeNumberRenew, n.ID, n.TMCreate, &commonaddress.Address{}, &commonaddress.Address{}); errBilling != nil {
		log.Errorf("Could not create a billing. number_id: %s", n.ID)
		return errors.Wrap(errBilling, "could not create a billing")
	}

	return nil
}
