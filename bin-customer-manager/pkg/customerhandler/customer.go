package customerhandler

import (
	"context"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// Delete deletes the customer.
func (h *customerHandler) Delete(ctx context.Context, id uuid.UUID) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Delete",
		"customer_id": id,
	})
	log.Debug("Deleteing the customer.")

	// get customer info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}

	if c.TMDelete != dbhandler.DefaultTimeStamp {
		// already deleted
		log.Infof("The customer already deleted. customer_id: %s", c.ID)
		return c, nil
	}

	res, err := h.dbDelete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the customer info. err: %v", err)
		return nil, err
	}

	return res, nil

}
