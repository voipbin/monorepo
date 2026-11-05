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
	// expiredRetention is how long an unverified customer stays in
	// StatusExpired, counted from the last update of the row (the expiry time
	// written by CleanupUnverified), before its PII is anonymized. It is also
	// the period in which the customer can recover the account with a new
	// verification email. It is independent of gracePeriod, which counts from
	// a freeze; both happen to be 30 days.
	expiredRetention = 30 * 24 * time.Hour

	// purgeBatchSize is the number of expired customers handled per run.
	purgeBatchSize = 100
)

// PurgeExpiredUnverified anonymizes expired customers that never verified their
// email address once expiredRetention has passed, and publishes a
// customer_deleted event for each customer it claims. The row is kept (status
// deleted, tm_delete set). Returns the number of customers actually purged.
//
// CustomerAnonymizeExpiredUnverified is a compare-and-set on status, email_verified
// and tm_delete, so a customer that verified after the list query, or that a
// concurrent replica already claimed, returns dbhandler.ErrNotFound. That is a
// routine outcome: it is not an error, not counted, and publishes nothing. Any
// other anonymize error is logged and skipped. A non-nil error is returned only
// when the list query fails.
//
// The event payload is built from the listed row instead of being read again.
// After the anonymize the row is deleted and is never listed again, so a failed
// read would lose the customer_deleted cascade for good. The subscribers use
// only the customer id.
func (h *customerHandler) PurgeExpiredUnverified(ctx context.Context) (int, error) {
	log := logrus.WithField("func", "PurgeExpiredUnverified")
	log.Debug("Running expired unverified customer purge.")

	cutoff := h.utilHandler.TimeNow().Add(-expiredRetention)

	customers, err := h.db.CustomerListExpiredUnverified(ctx, cutoff, purgeBatchSize)
	if err != nil {
		log.Errorf("Could not list expired unverified customers. err: %v", err)
		return 0, err
	}

	purged := 0
	for _, c := range customers {
		log.Infof("Purging expired unverified customer. customer_id: %s", c.ID)

		// anonymized identifiers use the first 8 chars of the UUID, like CleanupFrozenExpired
		shortID := c.ID.String()[:8]
		anonName := fmt.Sprintf("deleted_user_%s", shortID)
		anonEmail := fmt.Sprintf("deleted_%s@removed.voipbin.net", shortID)

		if err := h.db.CustomerAnonymizeExpiredUnverified(ctx, c.ID, anonName, anonEmail); err != nil {
			if errors.Is(err, dbhandler.ErrNotFound) {
				// verified after the list query, or claimed by a concurrent run
				log.Debugf("Expired unverified customer is no longer eligible. customer_id: %s", c.ID)
				continue
			}
			log.Errorf("Could not anonymize expired unverified customer. customer_id: %s, err: %v", c.ID, err)
			continue
		}

		// publish customer_deleted (cascading cleanup in the other services)
		// with the anonymized values; c is a pointer into the list, so copy it.
		ts := h.utilHandler.TimeNow()
		res := *c
		res.Name = anonName
		res.Email = anonEmail
		res.PhoneNumber = ""
		res.Address = ""
		res.WebhookURI = ""
		res.Status = customer.StatusDeleted
		res.TMDelete = ts
		res.TMUpdate = ts
		h.notifyHandler.PublishEvent(ctx, customer.EventTypeCustomerDeleted, &res)

		log.Infof("Purged expired unverified customer. customer_id: %s", c.ID)
		purged++
	}

	if purged > 0 {
		log.Infof("Expired unverified customer purge completed. purged: %d", purged)
	}

	return purged, nil
}
