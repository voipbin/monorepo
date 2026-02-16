package customerhandler

import (
	"context"
	"fmt"
	"time"

	"monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
)

const (
	expiryCheckInterval = 24 * time.Hour
	gracePeriod         = 30 * 24 * time.Hour // 30 days
)

// RunCleanupFrozenExpired periodically checks for frozen customers whose grace period has expired.
func (h *customerHandler) RunCleanupFrozenExpired(ctx context.Context) {
	log := logrus.WithField("func", "RunCleanupFrozenExpired")
	log.Info("Starting frozen customer expiry cleanup job.")

	ticker := time.NewTicker(expiryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping frozen customer expiry cleanup job.")
			return
		case <-ticker.C:
			h.cleanupFrozenExpired(ctx)
		}
	}
}

func (h *customerHandler) cleanupFrozenExpired(ctx context.Context) {
	log := logrus.WithField("func", "cleanupFrozenExpired")
	log.Debug("Running frozen customer expiry cleanup.")

	cutoff := time.Now().Add(-gracePeriod)

	customers, err := h.db.CustomerListFrozenExpired(ctx, cutoff)
	if err != nil {
		log.Errorf("Could not list frozen expired customers. err: %v", err)
		return
	}

	for _, c := range customers {
		log.Infof("Processing expired frozen customer. customer_id: %s, email: %s", c.ID, c.Email)

		// Generate anonymized identifiers using first 8 chars of UUID
		shortID := c.ID.String()[:8]
		anonName := fmt.Sprintf("deleted_user_%s", shortID)
		anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)

		if err := h.db.CustomerAnonymizePII(ctx, c.ID, anonName, anonEmail); err != nil {
			log.Errorf("Could not anonymize customer PII. customer_id: %s, err: %v", c.ID, err)
			continue
		}

		// Get updated customer after anonymization
		res, err := h.db.CustomerGet(ctx, c.ID)
		if err != nil {
			log.Errorf("Could not get anonymized customer. customer_id: %s, err: %v", c.ID, err)
			continue
		}

		// Publish customer_deleted event (reuse existing event type for cascading cleanup)
		h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerDeleted, res)
		log.Infof("Processed expired frozen customer. customer_id: %s", c.ID)
	}

	if len(customers) > 0 {
		log.Infof("Frozen expiry cleanup completed. processed: %d", len(customers))
	}
}
