package billinghandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Create creates a new billing and return the created billing.
func (h *billingHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	accountID uuid.UUID,
	referenceType billing.ReferenceType,
	referenceID uuid.UUID,
	costType billing.CostType,
	tmBillingStart *time.Time,
) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":             "Create",
		"customer_id":      customerID,
		"account_id":       accountID,
		"reference_type":   referenceType,
		"reference_id":     referenceID,
		"cost_type":        costType,
		"tm_billing_start": tmBillingStart,
	})

	tokenPerUnit, creditPerUnit := billing.GetCostInfo(costType)

	id := h.utilHandler.UUIDCreate()
	c := &billing.Billing{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		AccountID:         accountID,
		TransactionType:   billing.TransactionTypeUsage,
		Status:            billing.StatusProgressing,
		ReferenceType:     referenceType,
		ReferenceID:       referenceID,
		CostType:          costType,
		RateTokenPerUnit:  tokenPerUnit,
		RateCreditPerUnit: creditPerUnit,
		TMBillingStart:    tmBillingStart,
	}

	if errCreate := h.db.BillingCreate(ctx, c); errCreate != nil {
		log.Errorf("Could not create a billing. err: %v", errCreate)
		return nil, fmt.Errorf("could not create a billing. err: %v", errCreate)
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

	if res.ReferenceType != billing.ReferenceTypeCall && res.ReferenceType != billing.ReferenceTypeCallExtension {
		// if the billing's reference type is not a call type,
		// the result not valid.
		// because it is possible to billing has more than 2 billings of that reference id.
		// i.e. number type billing can have many of renewed billings.
		return nil, fmt.Errorf("wrong reference type")
	}

	return res, nil
}

// List returns a list of billings.
func (h *billingHandler) List(ctx context.Context, size uint64, token string, filters map[billing.Field]any) ([]*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "List",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	res, err := h.db.BillingList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not get billings. err: %v", err)
		return nil, errors.Wrap(err, "could not get billings")
	}

	return res, nil
}
