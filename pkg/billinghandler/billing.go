package billinghandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
)

// BillingStart starts the billing
func (h *billingHandler) BillingStart(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType billing.ReferenceType,
	referenceID uuid.UUID,
	tmBillingStart string,
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
		costPerUnit = defaultCostPerUnitReferenceTypeCall
		flagEnd = false

	case billing.ReferenceTypeSMS:
		costPerUnit = defaultCostPerUnitReferenceTypeSMS
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
			_ = h.BillingEnd(ctx, customerID, referenceType, referenceID, tmBillingStart, source, destination)
		}()
	}

	return nil
}

func (h *billingHandler) BillingEnd(
	ctx context.Context,
	customerID uuid.UUID,
	referenceType billing.ReferenceType,
	referenceID uuid.UUID,
	tmBillingEnd string,
	source *commonaddress.Address,
	destination *commonaddress.Address,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "BillingEnd",
		"customer_id":    customerID,
		"reference_type": referenceType,
		"reference_id":   referenceID,
		"tm_billing_end": tmBillingEnd,
	})

	// sleep
	log.Debugf("Sleeping before end the billing. reference_id: %s", referenceID)
	time.Sleep(time.Second * 3)

	// get billing
	b, err := h.GetByReferenceID(ctx, referenceID)
	if err != nil {
		// could not get billing. nothing to do.
		log.Errorf("Could not get billing. err: %v", err)
		return errors.Wrap(err, "could not get billing")
	}

	// calculate billing unit count
	var billingUnitCount time.Duration
	switch referenceType {
	case billing.ReferenceTypeCall:
		timeStart := h.utilHandler.TimeParse(b.TMBillingStart)
		timeEnd := h.utilHandler.TimeParse(tmBillingEnd)
		billingUnitCount = time.Duration(timeEnd.Sub(timeStart))

	case billing.ReferenceTypeSMS:
		billingUnitCount = 1
	}

	tmp, err := h.UpdateStatusEnd(ctx, b.ID, float32(billingUnitCount.Seconds()), tmBillingEnd)
	if err != nil {
		log.Errorf("Could not update billing status end. err: %v", err)
		return errors.Wrap(err, "could not update billing status end")
	}
	log.WithField("billing", tmp).Debugf("Updated billing status end. billing_id: %s", tmp.ID)

	// update account balance
	ac, err := h.accountHandler.SubtractBalance(ctx, tmp.CustomerID, tmp.CostTotal)
	if err != nil {
		log.Errorf("Could not substract the balance from the account. err: %v", err)
		return errors.Wrap(err, "could not substract the account balance from the account")
	}
	log.WithField("account", ac).Debugf("Updated account balance. account_id: %s, balance: %f", ac.ID, ac.Balance)

	return nil
}
