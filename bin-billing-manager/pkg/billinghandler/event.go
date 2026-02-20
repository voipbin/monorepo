package billinghandler

import (
	"context"
	"fmt"
	"strings"

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
	costType := getCostTypeForCall(c)

	if errBilling := h.BillingStart(ctx, c.CustomerID, refType, c.ID, costType, c.TMProgressing, &c.Source, &c.Destination); errBilling != nil {
		return errors.Wrap(errBilling, "could not start a billing")
	}

	return nil
}

// getReferenceTypeForCall determines the billing reference type based on the call's direction and address type.
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

// getCostTypeForCall determines the billing cost type based on the call's direction, source and destination.
func getCostTypeForCall(c *cmcall.Call) billing.CostType {
	switch c.Direction {
	case cmcall.DirectionIncoming:
		if c.Destination.Type == commonaddress.TypeTel {
			if strings.HasPrefix(c.Destination.Target, nmnumber.VirtualNumberPrefix) {
				return billing.CostTypeCallVN
			}
			if c.Source.Type == commonaddress.TypeTel {
				return billing.CostTypeCallPSTNIncoming
			}
		}
		if c.Source.Type == commonaddress.TypeSIP && c.Destination.Type == commonaddress.TypeExtension {
			return billing.CostTypeCallDirectExt
		}
		return billing.CostTypeCallExtension

	case cmcall.DirectionOutgoing:
		if c.Destination.Type == commonaddress.TypeTel {
			return billing.CostTypeCallPSTNOutgoing
		}
		return billing.CostTypeCallExtension

	default:
		return billing.CostTypeCallPSTNOutgoing
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
		if errBilling := h.BillingStart(ctx, m.CustomerID, billing.ReferenceTypeSMS, targetRefID, billing.CostTypeSMS, m.TMCreate, m.Source, &target.Destination); errBilling != nil {
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

	if errBilling := h.BillingStart(ctx, n.CustomerID, billing.ReferenceTypeNumber, n.ID, billing.CostTypeNumber, n.TMCreate, &commonaddress.Address{}, &commonaddress.Address{}); errBilling != nil {
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

	// Generate a deterministic reference ID per (number, year-month) so each monthly
	// renewal produces a unique billing record, while event redelivery within the same
	// month remains idempotent.
	currentYearMonth := h.utilHandler.TimeNow().Format("2006-01")
	referenceID := h.utilHandler.NewV5UUID(uuid.Nil, n.ID.String()+":renew:"+currentYearMonth)

	if errBilling := h.BillingStart(ctx, n.CustomerID, billing.ReferenceTypeNumberRenew, referenceID, billing.CostTypeNumberRenew, n.TMCreate, &commonaddress.Address{}, &commonaddress.Address{}); errBilling != nil {
		log.Errorf("Could not create a billing. number_id: %s", n.ID)
		return errors.Wrap(errBilling, "could not create a billing")
	}

	return nil
}
