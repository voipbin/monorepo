package allowancehandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// ProcessAllCycles iterates over all non-unlimited accounts and ensures they have a current
// allowance cycle. This is a safety net cron job — cycles are also created lazily on billing events.
func (h *allowanceHandler) ProcessAllCycles(ctx context.Context) error {
	log := logrus.WithField("func", "ProcessAllCycles")

	planTypes := []account.PlanType{
		account.PlanTypeFree,
		account.PlanTypeBasic,
		account.PlanTypeProfessional,
	}

	var errs []string
	for _, planType := range planTypes {
		if err := h.processAccountsForPlan(ctx, planType); err != nil {
			log.Errorf("Failed to process cycles for plan type. plan_type: %s, err: %v", planType, err)
			errs = append(errs, fmt.Sprintf("%s: %v", planType, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cycle processing errors: %s", strings.Join(errs, "; "))
	}

	return nil
}

// processAccountsForPlan processes all accounts of a given plan type.
func (h *allowanceHandler) processAccountsForPlan(ctx context.Context, planType account.PlanType) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "processAccountsForPlan",
		"plan_type": planType,
	})

	token := ""
	filters := map[account.Field]any{
		account.FieldPlanType: planType,
		account.FieldDeleted:  false,
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
			if _, err := h.EnsureCurrentCycle(ctx, acc.ID, acc.CustomerID, planType); err != nil {
				log.Errorf("Failed to ensure cycle for account. account_id: %s, err: %v", acc.ID, err)
			}
		}

		// Use last account's tm_create as token for next page.
		token = accounts[len(accounts)-1].TMCreate.Format(utilhandler.ISO8601Layout)
	}

	return nil
}
