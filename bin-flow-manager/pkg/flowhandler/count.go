package flowhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CountByCustomerID returns the count of active flows for the given customer.
func (h *flowHandler) CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.db.FlowCountByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get flow count. err: %v", err)
		return 0, fmt.Errorf("could not get flow count: %w", err)
	}

	return count, nil
}
