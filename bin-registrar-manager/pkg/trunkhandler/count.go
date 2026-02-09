package trunkhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CountByCustomerID returns the count of active trunks for the given customer.
func (h *trunkHandler) CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.db.TrunkCountByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get trunk count. err: %v", err)
		return 0, fmt.Errorf("could not get trunk count: %w", err)
	}

	return count, nil
}
