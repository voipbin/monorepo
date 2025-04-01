package billinghandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	mmmessage "monorepo/bin-message-manager/models/message"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCMCallProgressing handles the call-manager's call_progressing event
func (h *billingHandler) EventCMCallProgressing(ctx context.Context, c *cmcall.Call) error {
	if errBilling := h.BillingStart(ctx, c.CustomerID, billing.ReferenceTypeCall, c.ID, c.TMProgressing, &c.Source, &c.Destination); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *billingHandler) EventCMCallHangup(ctx context.Context, c *cmcall.Call) error {

	// get billing info
	b, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if c.TMHangup == "" || c.TMHangup == dbhandler.DefaultTimeStamp {
		return fmt.Errorf("invalid tm_hangup. call_id: %s, tm_hangup: %s", c.ID, c.TMHangup)
	}

	if errEnd := h.BillingEnd(ctx, b, c.TMHangup, &c.Source, &c.Destination); errEnd != nil {
		return errors.Wrapf(errEnd, "could not end the billing. billing_id: %s, call_id: %s, err: %v", b.ID, c.ID, errEnd)
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
			return errors.Wrapf(errBilling, "could not create a billing. target: %v", target)
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
		return errors.Wrapf(errBilling, "could not create a billing. number_id: %s", n.ID)
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
