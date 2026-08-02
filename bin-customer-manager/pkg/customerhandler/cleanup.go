package customerhandler

import (
	"context"
	"time"

	"monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

const (
	unverifiedMaxAge = time.Hour
)

// CleanupUnverified removes unverified customers older than unverifiedMaxAge.
// Returns the number of customers actually expired. A per-row CustomerUpdate
// failure is logged and does not abort the batch, but is excluded from the
// returned count. A non-nil error is returned only when the initial
// CustomerList call fails.
func (h *customerHandler) CleanupUnverified(ctx context.Context) (int, error) {
	log := logrus.WithField("func", "CleanupUnverified")
	log.Debug("Running unverified customer cleanup.")

	cutoff := time.Now().Add(-unverifiedMaxAge)
	cutoffStr := cutoff.Format("2006-01-02 15:04:05.000000")

	filters := map[customer.Field]any{
		customer.FieldEmailVerified: false,
		customer.FieldDeleted:       false,
	}

	customers, err := h.db.CustomerList(ctx, 100, cutoffStr, filters)
	if err != nil {
		log.Errorf("Could not list unverified customers. err: %v", err)
		return 0, err
	}

	expired := 0
	for _, c := range customers {
		log.Infof("Expiring unverified customer. customer_id: %s, email: %s", c.ID, c.Email)

		now := h.utilHandler.TimeNow()
		fields := map[customer.Field]any{
			customer.FieldStatus:   string(customer.StatusExpired),
			customer.FieldTMDelete: now,
		}
		if err := h.db.CustomerUpdate(ctx, c.ID, fields); err != nil {
			log.Errorf("Could not expire customer. customer_id: %s, err: %v", c.ID, err)
			continue
		}
		expired++
	}

	if expired > 0 {
		log.Infof("Cleanup completed. expired: %d", expired)
	}

	return expired, nil
}
