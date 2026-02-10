package billinghandler

import (
	"context"
	"fmt"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	mmmessage "monorepo/bin-message-manager/models/message"

	nmnumber "monorepo/bin-number-manager/models/number"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EventCMCallProgressing handles the call-manager's call_progressing event
func (h *billingHandler) EventCMCallProgressing(ctx context.Context, c *cmcall.Call) error {
	refType := getReferenceTypeForCall(c)

	if errBilling := h.BillingStart(ctx, c.CustomerID, refType, c.ID, c.TMProgressing, &c.Source, &c.Destination); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// getReferenceTypeForCall determines the billing reference type based on the call's direction and address type.
// Incoming calls with a PSTN source (TypeTel) are charged; outgoing calls with a PSTN destination (TypeTel) are charged.
// All other call types (extension, agent, sip, conference, line) are free (call_extension).
func getReferenceTypeForCall(c *cmcall.Call) billing.ReferenceType {
	switch c.Direction {
	case cmcall.DirectionIncoming:
		if c.Source.Type == commonaddress.TypeTel {
			return billing.ReferenceTypeCall
		}
		return billing.ReferenceTypeCallExtension

	case cmcall.DirectionOutgoing:
		if c.Destination.Type == commonaddress.TypeTel {
			return billing.ReferenceTypeCall
		}
		return billing.ReferenceTypeCallExtension

	default:
		// safe fallback: charge it
		return billing.ReferenceTypeCall
	}
}

// EventCMCallHangup handles the call-manager's call_hangup event
func (h *billingHandler) EventCMCallHangup(ctx context.Context, c *cmcall.Call) error {

	// get billing info
	b, err := h.GetByReferenceID(ctx, c.ID)
	if err != nil {
		// could not get billing. nothing to do.
		return nil
	}

	if c.TMHangup == nil {
		return fmt.Errorf("invalid tm_hangup. call_id: %s, tm_hangup: nil", c.ID)
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

	for i, target := range m.Targets {
		// Generate a deterministic per-target reference ID so each target gets its own
		// billing record, while event redelivery still triggers idempotency protection.
		targetRefID := uuid.NewV5(m.ID, fmt.Sprintf("target-%d", i))
		log.WithField("target", target).Debugf("Creating billing for message. destination: %v, target_ref_id: %s", target.Destination, targetRefID)
		if errBilling := h.BillingStart(ctx, m.CustomerID, billing.ReferenceTypeSMS, targetRefID, m.TMCreate, m.Source, &target.Destination); errBilling != nil {
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

	// virtual numbers have no billing
	if n.Type == nmnumber.TypeVirtual {
		log.Debugf("Skipping billing for virtual number. number_id: %s", n.ID)
		return nil
	}

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

	// virtual numbers have no billing
	if n.Type == nmnumber.TypeVirtual {
		log.Debugf("Skipping billing for virtual number. number_id: %s", n.ID)
		return nil
	}

	if errBilling := h.BillingStart(ctx, n.CustomerID, billing.ReferenceTypeNumberRenew, n.ID, n.TMCreate, &commonaddress.Address{}, &commonaddress.Address{}); errBilling != nil {
		log.Errorf("Could not create a billing. number_id: %s", n.ID)
		return errors.Wrap(errBilling, "could not create a billing")
	}

	return nil
}
