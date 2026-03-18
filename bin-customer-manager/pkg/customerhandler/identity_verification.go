package customerhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/customer"
)

// UpdateIdentityVerificationStatus updates the customer's identity verification status.
func (h *customerHandler) UpdateIdentityVerificationStatus(ctx context.Context, id uuid.UUID, status customer.IdentityVerificationStatus) (*customer.Customer, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "UpdateIdentityVerificationStatus",
		"customer_id": id,
		"status":      status,
	})
	log.Debug("Updating customer identity verification status.")

	if !status.IsValid() {
		return nil, fmt.Errorf("invalid identity verification status: %s", status)
	}

	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get customer info. err: %v", err)
		return nil, err
	}
	log.WithField("customer", c).Debugf("Retrieved customer info. customer_id: %s", c.ID)

	if c.IdentityVerificationStatus == status {
		log.Infof("Customer already has status %s. customer_id: %s", status, id)
		return c, nil
	}

	fields := map[customer.Field]any{
		customer.FieldIdentityVerificationStatus: string(status),
	}
	if err := h.db.CustomerUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update identity verification status. err: %v", err)
		return nil, err
	}

	res, err := h.db.CustomerGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated customer. err: %v", err)
		return nil, fmt.Errorf("could not get updated customer")
	}
	log.WithField("customer", res).Debugf("Retrieved updated customer info. customer_id: %s", res.ID)

	h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerIdentityVerificationUpdated, res)

	return res, nil
}
