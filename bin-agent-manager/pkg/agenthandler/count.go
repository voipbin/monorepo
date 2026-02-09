package agenthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CountByCustomerID returns the count of active agents for the given customer.
func (h *agentHandler) CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.db.AgentCountByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get agent count. err: %v", err)
		return 0, fmt.Errorf("could not get agent count: %w", err)
	}

	return count, nil
}
