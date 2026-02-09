package extensionhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// CountByCustomerID returns the count of active extensions for the given customer.
func (h *extensionHandler) CountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "CountByCustomerID",
		"customer_id": customerID,
	})

	count, err := h.dbBin.ExtensionCountByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get extension count. err: %v", err)
		return 0, fmt.Errorf("could not get extension count: %w", err)
	}

	return count, nil
}
