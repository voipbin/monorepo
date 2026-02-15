package allowancehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/allowance"
	"monorepo/bin-billing-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// GetCurrentCycle returns the active allowance cycle for the account.
func (h *allowanceHandler) GetCurrentCycle(ctx context.Context, accountID uuid.UUID) (*allowance.Allowance, error) {
	res, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// EnsureCurrentCycle returns the current cycle, creating it if it doesn't exist.
func (h *allowanceHandler) EnsureCurrentCycle(ctx context.Context, accountID uuid.UUID, customerID uuid.UUID, planType account.PlanType) (*allowance.Allowance, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "EnsureCurrentCycle",
		"account_id": accountID,
		"plan_type":  planType,
	})

	// try to get existing current cycle
	res, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err == nil {
		return res, nil
	}
	if err != dbhandler.ErrNotFound {
		return nil, fmt.Errorf("could not get current cycle. err: %v", err)
	}

	// no current cycle found — create one
	now := h.utilHandler.TimeNow()
	cycleStart, cycleEnd := computeCycleDates(*now)

	tokensTotal := account.PlanTokenMap[planType]

	a := &allowance.Allowance{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: customerID,
		},
		AccountID:   accountID,
		CycleStart:  &cycleStart,
		CycleEnd:    &cycleEnd,
		TokensTotal: tokensTotal,
		TokensUsed:  0,
	}

	if err := h.db.AllowanceCreate(ctx, a); err != nil {
		if err != dbhandler.ErrDuplicateKey {
			return nil, fmt.Errorf("could not create allowance cycle. err: %v", err)
		}
		// duplicate key = another process created the cycle concurrently — re-read it below
		log.Debugf("Allowance cycle already exists (concurrent creation). Re-reading. account_id: %s", accountID)
	} else {
		log.WithField("allowance", a).Debugf("Created new allowance cycle. allowance_id: %s", a.ID)
	}

	// re-read to get the authoritative row (either we just created it, or a concurrent process did)
	res, err = h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not get newly created cycle. err: %v", err)
	}

	return res, nil
}

// ConsumeTokens attempts to consume tokens from the current cycle. If tokens are
// insufficient, the overflow is charged as credits from the account balance.
// Returns the tokens consumed and the credit amount charged.
func (h *allowanceHandler) ConsumeTokens(ctx context.Context, accountID uuid.UUID, tokensNeeded int, creditPerUnit float32, tokenPerUnit int) (int, float32, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ConsumeTokens",
		"account_id":    accountID,
		"tokens_needed": tokensNeeded,
	})

	cycle, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return 0, 0, fmt.Errorf("could not get current cycle. err: %v", err)
	}

	tokensConsumed, creditCharged, err := h.db.AllowanceConsumeTokens(ctx, cycle.ID, accountID, tokensNeeded, creditPerUnit, tokenPerUnit)
	if err != nil {
		return 0, 0, fmt.Errorf("could not consume tokens. err: %v", err)
	}
	log.Debugf("Token consumption complete. consumed: %d, credit_charged: %f", tokensConsumed, creditCharged)

	return tokensConsumed, creditCharged, nil
}

// ListByAccountID returns a paginated list of allowance cycles for an account.
func (h *allowanceHandler) ListByAccountID(ctx context.Context, accountID uuid.UUID, size uint64, token string) ([]*allowance.Allowance, error) {
	filters := map[allowance.Field]any{
		allowance.FieldAccountID: accountID,
		allowance.FieldDeleted:   false,
	}

	return h.db.AllowanceList(ctx, size, token, filters)
}

// AddTokens adds tokens to the current cycle's total allocation.
func (h *allowanceHandler) AddTokens(ctx context.Context, accountID uuid.UUID, amount int) (*allowance.Allowance, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddTokens",
		"account_id": accountID,
		"amount":     amount,
	})

	cycle, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not get current cycle. err: %v", err)
	}

	newTotal := cycle.TokensTotal + amount
	if err := h.db.AllowanceUpdate(ctx, cycle.ID, map[allowance.Field]any{
		allowance.FieldTokensTotal: newTotal,
	}); err != nil {
		return nil, fmt.Errorf("could not update tokens_total. err: %v", err)
	}
	log.Debugf("Added tokens. allowance_id: %s, old_total: %d, new_total: %d", cycle.ID, cycle.TokensTotal, newTotal)

	return h.db.AllowanceGet(ctx, cycle.ID)
}

// SubtractTokens subtracts tokens from the current cycle's total allocation.
func (h *allowanceHandler) SubtractTokens(ctx context.Context, accountID uuid.UUID, amount int) (*allowance.Allowance, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SubtractTokens",
		"account_id": accountID,
		"amount":     amount,
	})

	cycle, err := h.db.AllowanceGetCurrentByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("could not get current cycle. err: %v", err)
	}

	newTotal := cycle.TokensTotal - amount
	if newTotal < 0 {
		return nil, fmt.Errorf("cannot subtract %d tokens: current total is %d", amount, cycle.TokensTotal)
	}

	if err := h.db.AllowanceUpdate(ctx, cycle.ID, map[allowance.Field]any{
		allowance.FieldTokensTotal: newTotal,
	}); err != nil {
		return nil, fmt.Errorf("could not update tokens_total. err: %v", err)
	}
	log.Debugf("Subtracted tokens. allowance_id: %s, old_total: %d, new_total: %d", cycle.ID, cycle.TokensTotal, newTotal)

	return h.db.AllowanceGet(ctx, cycle.ID)
}

// computeCycleDates calculates the cycle start (beginning of current month) and
// cycle end (beginning of next month) from the given time.
func computeCycleDates(now time.Time) (time.Time, time.Time) {
	cycleStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	cycleEnd := cycleStart.AddDate(0, 1, 0)
	return cycleStart, cycleEnd
}
