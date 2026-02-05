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
)

// BillingStart starts the billing
func (h *billingHandler) BillingStart(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType billing.ReferenceType,
	referenceID uuid.UUID,
	tmBillingStart *time.Time,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":             "BillingStart",
		"customer_id":      customerID,
		"reference_type":   referenceType,
		"reference_id":     referenceID,
		"tm_billing_start": tmBillingStart,
	})

	// get account
	a, err := h.accountHandler.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return errors.Wrap(err, "could not get account info")
	}

	// create billing
	var costPerUnit float32
	var flagEnd bool
	switch referenceType {
	case billing.ReferenceTypeCall:
		costPerUnit = billing.DefaultCostPerUnitReferenceTypeCall
		flagEnd = false

	case billing.ReferenceTypeSMS:
		costPerUnit = billing.DefaultCostPerUnitReferenceTypeSMS
		flagEnd = true

	case billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		costPerUnit = billing.DefaultCostPerUnitReferenceTypeNumber
		flagEnd = true

	default:
		log.Errorf("Unsupported reference type. reference_type: %s", referenceType)
		return fmt.Errorf("unsupported reference type")
	}

	tmp, err := h.Create(ctx, a.CustomerID, a.ID, referenceType, referenceID, costPerUnit, tmBillingStart)
	if err != nil {
		log.Errorf("Could not create a billing. err: %v", err)
		return errors.Wrap(err, "could not create a billing")
	}
	log.WithField("billing", tmp).Debugf("Created a new billing. billing_id: %s", tmp.ID)

	if flagEnd {
		log.Debugf("The end flag has set. End the billing now. reference_id: %s", referenceID)
		go func() {
			if errBilling := h.BillingEnd(context.Background(), tmp, tmBillingStart, source, destination); errBilling != nil {
				// note: we could not bill the cost. But we write the log only here.
				log.Errorf("Could not end the billing. err: %v", errBilling)
			}
		}()
	}

	return nil
}

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
		"tm_billing_end": tmBillingEnd,
	})

	// calculate billing unit count
	var billingUnitCount time.Duration
	switch bill.ReferenceType {
	case billing.ReferenceTypeCall:
		if bill.TMBillingStart != nil && tmBillingEnd != nil {
			billingUnitCount = time.Duration(tmBillingEnd.Sub(*bill.TMBillingStart))
		}

	case billing.ReferenceTypeSMS, billing.ReferenceTypeNumber, billing.ReferenceTypeNumberRenew:
		billingUnitCount = time.Duration(time.Second * 1)

	default:
		log.WithField("billing", bill).Errorf("Unsupported billing reference type. reference_type: %s", bill.ReferenceType)
		return fmt.Errorf("unsupported billing reference type. reference_type: %s", bill.ReferenceType)
	}

	tmp, err := h.UpdateStatusEnd(ctx, bill.ID, float32(billingUnitCount.Seconds()), tmBillingEnd)
	if err != nil {
		log.Errorf("Could not update billing status end. err: %v", err)
		return errors.Wrap(err, "could not update billing status end")
	}
	log.WithField("billing", tmp).Debugf("Updated billing status end. billing_id: %s", tmp.ID)

	// update account balance
	ac, err := h.accountHandler.SubtractBalance(ctx, tmp.AccountID, tmp.CostTotal)
	if err != nil {
		log.Errorf("Could not substract the balance from the account. err: %v", err)
		return errors.Wrap(err, "could not substract the account balance from the account")
	}
	log.WithField("account", ac).Debugf("Updated account balance. account_id: %s, balance: %f", ac.ID, ac.Balance)

	return nil
}
