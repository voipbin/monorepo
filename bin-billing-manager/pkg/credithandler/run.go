package credithandler

import (
	"context"
	"fmt"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
)

// ProcessAll iterates over all free-tier accounts and processes monthly credit top-ups.
func (h *handler) ProcessAll(ctx context.Context) error {
	log := logrus.WithField("func", "ProcessAll")

	token := ""
	filters := map[account.Field]any{
		account.FieldPlanType: account.PlanTypeFree,
	}

	for {
		accounts, err := h.db.AccountList(ctx, 100, token, filters)
		if err != nil {
			return fmt.Errorf("could not list accounts. err: %v", err)
		}
		if len(accounts) == 0 {
			break
		}

		for _, acc := range accounts {
			if err := h.processAccount(ctx, acc); err != nil {
				log.Errorf("Failed to process credit for account. account_id: %s, err: %v", acc.ID, err)
				// continue to next account — don't block on individual failures
			}
		}

		// Use last account's tm_create as token for next page.
		// AccountList expects ISO8601Layout format (used by utilHandler.TimeGetCurTime).
		token = accounts[len(accounts)-1].TMCreate.Format(utilhandler.ISO8601Layout)
	}

	return nil
}
