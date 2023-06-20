package customerhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// IsValidBalance returns true if the customer's billing account has enough balance
func (h *customerHandler) IsValidBalance(ctx context.Context, customerID uuid.UUID) (bool, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "IsValidBalance",
		"customer_id": customerID,
	})

	// get customer info
	c, err := h.Get(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return false, errors.Wrap(err, "could not get customer info")
	}

	// get account info
	a, err := h.reqHandler.BillingV1AccountGet(ctx, c.BillingAccountID)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return false, errors.Wrap(err, "could not get account info")
	}

	if a.Balance <= 0 {
		return false, nil
	}

	return true, nil
}
