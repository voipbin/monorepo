package customerhandler

import (
	"context"
	"time"

	"monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

const (
	cleanupInterval   = 15 * time.Minute
	unverifiedMaxAge  = time.Hour
)

// RunCleanupUnverified periodically removes unverified customers older than maxAge.
func (h *customerHandler) RunCleanupUnverified(ctx context.Context) {
	log := logrus.WithField("func", "RunCleanupUnverified")
	log.Info("Starting unverified customer cleanup job.")

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping unverified customer cleanup job.")
			return
		case <-ticker.C:
			h.cleanupUnverified(ctx)
		}
	}
}

func (h *customerHandler) cleanupUnverified(ctx context.Context) {
	log := logrus.WithField("func", "cleanupUnverified")
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
		return
	}

	for _, c := range customers {
		log.Infof("Deleting expired unverified customer. customer_id: %s, email: %s", c.ID, c.Email)
		if err := h.db.CustomerHardDelete(ctx, c.ID); err != nil {
			log.Errorf("Could not hard-delete customer. customer_id: %s, err: %v", c.ID, err)
		}
	}

	if len(customers) > 0 {
		log.Infof("Cleanup completed. deleted: %d", len(customers))
	}
}
