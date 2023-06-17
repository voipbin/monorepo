package billinghandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/dbhandler"
)

// Create creates a new billing and return the created billing.
func (h *billingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	accountID uuid.UUID,
	referenceType billing.ReferenceType,
	referenceID uuid.UUID,
	costPerUnit float32,
	tmBillingStart string,
) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Create",
		"customer_id":      customerID,
		"account_id":       accountID,
		"reference_type":   referenceType,
		"reference_id":     referenceID,
		"cost_per_second":  costPerUnit,
		"tm_billing_start": tmBillingStart,
	})

	id := h.utilHandler.UUIDCreate()
	c := &billing.Billing{
		ID:               id,
		CustomerID:       customerID,
		AccountID:        accountID,
		Status:           billing.StatusProgressing,
		ReferenceType:    referenceType,
		ReferenceID:      referenceID,
		CostPerUnit:      costPerUnit,
		CostTotal:        0,
		BillingUnitCount: 0,
		TMBillingStart:   tmBillingStart,
		TMBillingEnd:     dbhandler.DefaultTimeStamp,
	}

	if errCreate := h.db.BillingCreate(ctx, c); errCreate != nil {
		log.Errorf("Could not create a billing. err: %v", errCreate)
	}
	promBillingCreateTotal.WithLabelValues(string(c.ReferenceType)).Inc()

	res, err := h.db.BillingGet(ctx, c.ID)
	if err != nil {
		log.Errorf("Could not get a created billing. err: %v", err)
		return nil, err
	}
	h.notifyHandler.PublishEvent(ctx, billing.EventTypeBillingCreated, res)

	return res, nil
}

// CreateByCall creates a new billing of the given call and return the created billing.
func (h *billingHandler) CreateByCall(ctx context.Context, c *cmcall.Call) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "CreateByCall",
		"call": c,
	})

	// get account
	a, err := h.accountHandler.GetByCustomerID(ctx, c.CustomerID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return nil, errors.Wrap(err, "could not get account info")
	}

	res, err := h.Create(ctx, c.CustomerID, a.ID, billing.ReferenceTypeCall, c.ID, defaultCostPerUnitReferenceTypeCall, c.TMProgressing)
	if err != nil {
		log.Errorf("Could not create a billing. err: %v", err)
		return nil, errors.Wrap(err, "could not create a billing")
	}

	return res, nil
}

// Get returns a billing.
func (h *billingHandler) Get(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Get",
		"billing_id": id,
	})

	res, err := h.db.BillingGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get billing. err: %v", err)
		return nil, errors.Wrap(err, "could not get billing")
	}

	return res, nil
}

// GetByReferenceID returns a billing of the given reference id.
func (h *billingHandler) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetByReferenceID",
		"billing_id": referenceID,
	})

	res, err := h.db.BillingGetByReferenceID(ctx, referenceID)
	if err != nil {
		log.Errorf("Could not get billing. err: %v", err)
		return nil, errors.Wrap(err, "could not get billing")
	}

	return res, nil
}

// Gets returns a list of billings.
func (h *billingHandler) Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Gets",
		"customer_id": customerID,
		"size":        size,
		"token":       token,
	})

	res, err := h.db.BillingGetsByCustomerID(ctx, customerID, size, token)
	if err != nil {
		log.Errorf("Could not get billings. err: %v", err)
		return nil, errors.Wrap(err, "could not get billings")
	}

	return res, nil
}

// UpdateStatusEnd creats a new billing and return the created billing.
func (h *billingHandler) UpdateStatusEnd(ctx context.Context, id uuid.UUID, billingUnitCount float32, tmBillingEnd string) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "UpdateStatusEnd",
		"billing_id":       id,
		"billing_duration": billingUnitCount,
		"tm_billing_end":   tmBillingEnd,
	})

	if errSet := h.db.BillingSetStatusEnd(ctx, id, billingUnitCount, tmBillingEnd); errSet != nil {
		log.Errorf("Could not set status to end. err: %v", errSet)
		return nil, errors.Wrap(errSet, "could not set status to end")
	}

	res, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated billing info. err: %v", err)
		return nil, errors.Wrap(err, "could not get updated billing info")
	}

	promBillingEndTotal.WithLabelValues(string(res.ReferenceType)).Inc()

	return res, nil
}
