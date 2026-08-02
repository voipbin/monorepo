package customerhandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/dbhandler"

	"github.com/sirupsen/logrus"
)

const (
	gracePeriod = 30 * 24 * time.Hour // 30 days
)

// CleanupFrozenExpired processes frozen customers whose grace period has
// expired: it anonymizes PII and publishes a customer_deleted event for each
// one it successfully claims. Returns the number of customers actually
// processed.
//
// CustomerAnonymizePII now guards on status='frozen' AND tm_delete IS NULL
// (design §5.5), so a 0-rows result (dbhandler.ErrNotFound) is an expected,
// routine outcome under concurrent replicas racing the same row — it is not
// logged as an error, not counted as processed, and does not publish an
// event. Any other error is still logged and skipped, unchanged from before.
// A non-nil error is returned only when CustomerListFrozenExpired fails.
func (h *customerHandler) CleanupFrozenExpired(ctx context.Context) (int, error) {
	log := logrus.WithField("func", "CleanupFrozenExpired")
	log.Debug("Running frozen customer expiry cleanup.")

	cutoff := time.Now().Add(-gracePeriod)

	customers, err := h.db.CustomerListFrozenExpired(ctx, cutoff)
	if err != nil {
		log.Errorf("Could not list frozen expired customers. err: %v", err)
		return 0, err
	}

	processed := 0
	for _, c := range customers {
		log.Infof("Processing expired frozen customer. customer_id: %s, email: %s", c.ID, c.Email)

		// Generate anonymized identifiers using first 8 chars of UUID
		shortID := c.ID.String()[:8]
		anonName := fmt.Sprintf("deleted_user_%s", shortID)
		anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)

		if err := h.db.CustomerAnonymizePII(ctx, c.ID, anonName, anonEmail); err != nil {
			if errors.Is(err, dbhandler.ErrNotFound) {
				// A concurrent claimant already won the guard (status/tm_delete
				// no longer matched) — routine CAS-skip, not a failure.
				log.Debugf("Frozen customer already claimed by a concurrent run. customer_id: %s", c.ID)
				continue
			}
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
		processed++
	}

	if processed > 0 {
		log.Infof("Frozen expiry cleanup completed. processed: %d", processed)
	}

	return processed, nil
}
