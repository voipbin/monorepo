package customerhandler

import (
	"context"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonbilling "monorepo/bin-common-handler/models/billing"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// IsValidBalance returns true if the customer's billing account has enough balance
func (h *customerHandler) IsValidBalance(ctx context.Context, customerID uuid.UUID, billingType bmbilling.ReferenceType, country string, count int) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":         "IsValidBalance",
		"customer_id":  customerID,
		"billing_type": billingType,
		"country":      country,
		"count":        count,
	})

	// get customer info
	c, err := h.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return false, errors.Wrap(err, "could not get customer info")
	}

	if count < 1 {
		count = 1
	}

	//
	valid, err := h.reqHandler.BillingV1AccountIsValidBalance(ctx, c.BillingAccountID, billingType, country, count)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, errors.Wrap(err, "could not get account info")
	}

	return valid, nil
}

// IsValidResourceLimit returns true if the customer's billing account has not exceeded the resource limit for the given resource type.
func (h *customerHandler) IsValidResourceLimit(ctx context.Context, customerID uuid.UUID, resourceType commonbilling.ResourceType) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "IsValidResourceLimit",
		"customer_id":   customerID,
		"resource_type": resourceType,
	})

	c, err := h.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return false, errors.Wrap(err, "could not get customer info")
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	valid, err := h.reqHandler.BillingV1AccountIsValidResourceLimit(ctx, c.BillingAccountID, resourceType)
	if err != nil {
		log.Errorf("Could not validate the account's resource limit. err: %v", err)
		return false, errors.Wrap(err, "could not validate the account's resource limit")
	}

	return valid, nil
}
