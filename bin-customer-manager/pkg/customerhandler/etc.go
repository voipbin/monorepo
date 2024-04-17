package customerhandler

import (
	"context"

	bmbilling "monorepo/bin-billing-manager/models/billing"

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
