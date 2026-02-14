package billinghandler

import (
	"context"
	"fmt"
	"math"
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
		case billing.ReferenceTypeSMS, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
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
	case billing.ReferenceTypeSMS:
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
// and updating the billing record.
//
// NOTE: Token consumption (via AllowanceConsumeTokens) and the billing record update
// (via UpdateStatusEnd) are NOT in a single database transaction. This is a deliberate
// trade-off: AllowanceConsumeTokens locks both the allowance and account rows in one
// transaction, and the billing record update is a separate write. If the billing update
// fails after tokens are consumed, the tokens are "lost" for that cycle. This is
// acceptable because:
//   - The event will be retried by the message broker (RabbitMQ redelivery), which will
//     see the billing record still in progressing status and re-run BillingEnd.
//   - The idempotency check in BillingStart prevents duplicate billing creation.
//   - In the worst case, a small number of tokens are consumed without a matching billing
//     record — this is preferable to holding long-lived locks across two tables.
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

	// calculate cost unit count (minutes for calls, 1 for SMS/number)
	var costUnitCount float32
	switch bill.ReferenceType {
	case billing.ReferenceTypeCall, billing.ReferenceTypeCallExtension:
		if bill.TMBillingStart != nil && tmBillingEnd != nil {
			durationSec := tmBillingEnd.Sub(*bill.TMBillingStart).Seconds()
			costUnitCount = float32(math.Ceil(durationSec / 60.0))
		}

	case billing.ReferenceTypeSMS, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		costUnitCount = 1

	default:
		log.WithField("billing", bill).Errorf("Unsupported billing reference type. reference_type: %s", bill.ReferenceType)
		return fmt.Errorf("unsupported billing reference type. reference_type: %s", bill.ReferenceType)
	}

	var costTokenTotal int
	var costCreditTotal float32

	tokenPerUnit := bill.CostTokenPerUnit
	creditPerUnit := bill.CostCreditPerUnit

	switch {
	case bill.CostType == billing.CostTypeCallExtension:
		// Free — no tokens, no credits
		costTokenTotal = 0
		costCreditTotal = 0

	case tokenPerUnit > 0:
		// Token-eligible (VN call, SMS)
		tokensNeeded := int(costUnitCount) * tokenPerUnit

		tokensConsumed, creditCharged, err := h.allowanceHandler.ConsumeTokens(ctx, bill.AccountID, tokensNeeded, creditPerUnit, tokenPerUnit)
		if err != nil {
			log.Errorf("Could not consume tokens. Reverting billing status. err: %v", err)
			if revertErr := h.db.BillingSetStatus(ctx, bill.ID, billing.StatusProgressing); revertErr != nil {
				log.Errorf("Could not revert billing status to progressing. billing_id: %s, err: %v", bill.ID, revertErr)
			}
			return errors.Wrap(err, "could not consume tokens")
		}
		costTokenTotal = tokensConsumed
		costCreditTotal = creditCharged

	case creditPerUnit > 0:
		// Credit-only (PSTN calls, number purchases)
		costCreditTotal = costUnitCount * creditPerUnit

		if costCreditTotal > 0 {
			if _, err := h.accountHandler.SubtractBalanceWithCheck(ctx, bill.AccountID, costCreditTotal); err != nil {
				log.Errorf("Could not subtract the balance from the account. Reverting billing status. err: %v", err)
				if revertErr := h.db.BillingSetStatus(ctx, bill.ID, billing.StatusProgressing); revertErr != nil {
					log.Errorf("Could not revert billing status to progressing. billing_id: %s, err: %v", bill.ID, revertErr)
				}
				return errors.Wrap(err, "could not subtract the account balance from the account")
			}
		}
		costTokenTotal = 0
	}

	tmp, err := h.UpdateStatusEnd(ctx, bill.ID, costUnitCount, costTokenTotal, costCreditTotal, tmBillingEnd)
	if err != nil {
		log.Errorf("Could not update billing status end. err: %v", err)
		return errors.Wrap(err, "could not update billing status end")
	}
	log.WithField("billing", tmp).Debugf("Updated billing status end. billing_id: %s", tmp.ID)

	return nil
}
