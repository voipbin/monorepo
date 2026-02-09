package queuehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CountByCustomerID returns the count of active queues for the given customer.
func (h *queueHandler) CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.db.QueueCountByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get queue count. err: %v", err)
		return 0, fmt.Errorf("could not get queue count: %w", err)
	}

	return count, nil
}
