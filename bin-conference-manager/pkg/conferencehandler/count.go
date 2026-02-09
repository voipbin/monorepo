package conferencehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CountByCustomerID returns the count of active conferences for the given customer.
func (h *conferenceHandler) CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.db.ConferenceCountByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get conference count. err: %v", err)
		return 0, fmt.Errorf("could not get conference count: %w", err)
	}

	return count, nil
}
