package credithandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
)

// processAccount processes a single free-tier account for monthly credit top-up.
func (h *handler) processAccount(ctx context.Context, acc *account.Account) error {
	log := logrus.WithFields(logrus.Fields{"func": "processAccount", "account_id": acc.ID})

	// Generate deterministic reference_id for this account + month.
	// All pods produce the same UUID for the same account and month,
	// so the unique index on (reference_type, reference_id) prevents
	// duplicate top-ups without expensive row-level locking.
	now := h.utilHandler.TimeNow()
	currentYearMonth := now.Format("2006-01")
	referenceID := h.utilHandler.NewV5UUID(uuid.Nil, acc.ID.String()+":"+currentYearMonth)
	b := &billing.Billing{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: acc.CustomerID,
		},
		AccountID:        acc.ID,
		ReferenceType:    billing.ReferenceTypeCreditFreeTier,
		ReferenceID:      referenceID,
		CostPerUnit:      0,
		CostTotal:        0, // updated inside transaction if credit is needed
		BillingUnitCount: 1.0,
		Status:           billing.StatusEnd,
		TMBillingStart:   now,
		TMBillingEnd:     now,
	}

	created, err := h.db.BillingCreditTopUp(ctx, b, acc.ID, FreeTierCreditAmount)
	if err != nil {
		return fmt.Errorf("could not top up credit. err: %v", err)
	}

	if created {
		log.Debugf("Credit top-up processed. account_id: %s", acc.ID)
	}

	return nil
}
