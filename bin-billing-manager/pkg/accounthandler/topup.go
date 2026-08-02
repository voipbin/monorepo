package accounthandler

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
)

// TopUpDue pages through all non-deleted accounts and applies the monthly token
// top-up to every account whose tm_next_topup is due. A per-row error is logged and
// counted as failed without aborting the batch. A CAS-skip (AccountTopUpTokens
// returning applied=false with a nil error — the account was already topped up this
// cycle by a concurrent claimant) increments neither counter. Only a failure of the
// AccountList call itself (nothing could be attempted) returns a non-nil error.
func (h *accountHandler) TopUpDue(ctx context.Context) (int, int, error) {
	log := logrus.WithField("func", "TopUpDue")

	now := h.utilHandler.TimeNow()
	filters := map[account.Field]any{
		account.FieldDeleted: false,
	}

	processed := 0
	failed := 0

	// Paginate through all accounts
	var pageToken string
	for {
		accounts, err := h.db.AccountList(ctx, 500, pageToken, filters)
		if err != nil {
			return processed, failed, fmt.Errorf("TopUpDue: could not list accounts. err: %v", err)
		}
		if len(accounts) == 0 {
			break
		}

		for _, a := range accounts {
			if a.TmNextTopUp == nil || a.TmNextTopUp.After(*now) {
				continue
			}

			tokenAmount, ok := account.PlanTokenMap[a.PlanType]
			if !ok || tokenAmount <= 0 {
				continue
			}

			applied, errTopUp := h.db.AccountTopUpTokens(ctx, a.ID, a.CustomerID, tokenAmount, string(a.PlanType))
			if errTopUp != nil {
				log.Errorf("Could not top up tokens for account. account_id: %s, err: %v", a.ID, errTopUp)
				failed++
				continue
			}
			if !applied {
				// CAS-skip: already topped up this cycle by a concurrent claimant.
				// Neither processed nor failed.
				continue
			}

			processed++
			log.Infof("Topped up tokens for account. account_id: %s, tokens: %d", a.ID, tokenAmount)
		}

		if len(accounts) < 500 {
			break
		}
		pageToken = accounts[len(accounts)-1].TMCreate.Format(time.RFC3339Nano)
	}

	return processed, failed, nil
}
