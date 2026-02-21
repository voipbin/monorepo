package billinghandler

import (
	"context"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// BillingStart starts the billing
func (h *billingHandler) BillingStart(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType billing.ReferenceType,
	referenceID uuid.UUID,
	costType billing.CostType,
	tmBillingStart *time.Time,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":             "BillingStart",
		"customer_id":      customerID,
		"reference_type":   referenceType,
		"reference_id":     referenceID,
		"cost_type":        costType,
		"tm_billing_start": tmBillingStart,
	})

	// idempotency check — return early if billing already exists for this reference
	existing, err := h.db.BillingGetByReferenceTypeAndID(ctx, referenceType, referenceID)
	if err != nil && err != dbhandler.ErrNotFound {
		// real DB error (connection timeout, etc.) — do NOT fall through to create
		return errors.Wrap(err, "could not check for existing billing")
	}
	if err == nil && existing != nil {
		if existing.Status == billing.StatusEnd {
			log.WithField("billing", existing).Debugf("Billing already completed. Skipping. reference_type: %s, reference_id: %s", referenceType, referenceID)
			return nil
		}
		// billing exists but not completed — re-run end phase for immediate-end types
		log.WithField("billing", existing).Debugf("Billing exists but not completed. status: %s", existing.Status)
		switch referenceType {
		case billing.ReferenceTypeSMS, billing.ReferenceTypeEmail, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
			if errBilling := h.BillingEnd(ctx, existing, tmBillingStart, source, destination); errBilling != nil {
				return errors.Wrap(errBilling, "could not complete billing on retry")
			}
		}
		return nil
	}

	// get account
	a, err := h.accountHandler.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return errors.Wrap(err, "could not get account info")
	}

	// determine if this is an immediate-end type
	var flagEnd bool
	switch referenceType {
	case billing.ReferenceTypeCall, billing.ReferenceTypeCallExtension:
		flagEnd = false
	case billing.ReferenceTypeSMS, billing.ReferenceTypeEmail:
		flagEnd = true
	case billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		flagEnd = true
	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return fmt.Errorf("unsupported reference type")
	}

	tmp, err := h.Create(ctx, a.CustomerID, a.ID, referenceType, referenceID, costType, tmBillingStart)
	if err != nil {
		log.Errorf("Could not create a billing. err: %v", err)
		return errors.Wrap(err, "could not create a billing")
	}
	log.WithField("billing", tmp).Debugf("Created a new billing. billing_id: %s", tmp.ID)

	if flagEnd {
		log.Debugf("The end flag has set. End the billing now. reference_id: %s", referenceID)
		if errBilling := h.BillingEnd(ctx, tmp, tmBillingStart, source, destination); errBilling != nil {
			log.Errorf("Could not end the billing. err: %v", errBilling)
			return errors.Wrap(errBilling, "could not end the billing")
		}
	}

	return nil
}

// BillingEnd finalises a billing record by calculating costs, consuming tokens/credits,
// and updating the billing record atomically using BillingConsumeAndRecord.
func (h *billingHandler) BillingEnd(
	ctx context.Context,
	bill *billing.Billing,
	tmBillingEnd *time.Time,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "BillingEnd",
		"id":             bill.ID,
		"cost_type":      bill.CostType,
		"tm_billing_end": tmBillingEnd,
	})

	// Extension calls are free — no billing needed
	if bill.CostType == billing.CostTypeCallExtension || bill.CostType == billing.CostTypeCallDirectExt {
		// Just mark as end with zero costs
		if err := h.db.BillingSetStatusEnd(ctx, bill.ID, 0, 0, 0, 0, 0, 0, tmBillingEnd); err != nil {
			log.Errorf("Could not set status to end for free call. err: %v", err)
			return fmt.Errorf("could not end free billing. err: %v", err)
		}

		res, err := h.db.BillingGet(ctx, bill.ID)
		if err != nil {
			log.Errorf("Could not get updated billing. err: %v", err)
			return err
		}

		promBillingEndTotal.WithLabelValues(string(res.ReferenceType)).Inc()
		if res.TMBillingStart != nil && res.TMBillingEnd != nil {
			promBillingDurationSeconds.WithLabelValues(string(res.ReferenceType)).Observe(res.TMBillingEnd.Sub(*res.TMBillingStart).Seconds())
		}
		h.notifyHandler.PublishEvent(ctx, billing.EventTypeBillingUpdated, res)
		return nil
	}

	// Calculate usage duration and billable units
	var usageDuration int
	var billableUnits int
	switch bill.ReferenceType {
	case billing.ReferenceTypeCall, billing.ReferenceTypeCallExtension:
		if bill.TMBillingStart != nil && tmBillingEnd != nil {
			usageDuration = int(tmBillingEnd.Sub(*bill.TMBillingStart).Seconds())
			billableUnits = billing.CalculateBillableUnits(usageDuration)
		}
	case billing.ReferenceTypeSMS, billing.ReferenceTypeEmail, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		usageDuration = 0
		billableUnits = 1
	default:
		log.Errorf("Unsupported billing reference type. reference_type: %s", bill.ReferenceType)
		return fmt.Errorf("unsupported billing reference type. reference_type: %s", bill.ReferenceType)
	}

	// Use atomic consume-and-record transaction
	costInfo := billing.GetCostInfo(bill.CostType)
	res, err := h.db.BillingConsumeAndRecord(ctx, bill, bill.AccountID, billableUnits, usageDuration, costInfo, tmBillingEnd)
	if err != nil {
		log.Errorf("Could not consume and record billing. err: %v", err)
		return fmt.Errorf("could not consume and record billing. err: %v", err)
	}
	log.WithField("billing", res).Debugf("Billing consumed and recorded. billing_id: %s", res.ID)

	promBillingEndTotal.WithLabelValues(string(res.ReferenceType)).Inc()
	if res.TMBillingStart != nil && res.TMBillingEnd != nil {
		promBillingDurationSeconds.WithLabelValues(string(res.ReferenceType)).Observe(res.TMBillingEnd.Sub(*res.TMBillingStart).Seconds())
	}
	h.notifyHandler.PublishEvent(ctx, billing.EventTypeBillingUpdated, res)

	return nil
}
