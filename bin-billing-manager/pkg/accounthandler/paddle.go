package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// checkPaddleIdempotency checks if a billing record with the given event ID already exists.
// Uses a single DB query on the idempotency_key column.
func (h *accountHandler) checkPaddleIdempotency(ctx context.Context, eventID string) (bool, error) {
	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	_, err := h.db.BillingGetByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		return true, nil // Already processed
	}
	if err == dbhandler.ErrNotFound {
		return false, nil // Not processed yet
	}
	return false, fmt.Errorf("could not check idempotency: %w", err)
}

// PaddleCreditTopUp adds credit balance from a Paddle credit purchase.
// Uses AccountPaddleAddCredit for atomic balance+billing in one transaction (no double-ledger).
func (h *accountHandler) PaddleCreditTopUp(ctx context.Context, customerID uuid.UUID, amountCreditMicros int64, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PaddleCreditTopUp",
		"customer_id": customerID,
		"amount":      amountCreditMicros,
		"event_id":    eventID,
	})

	if amountCreditMicros <= 0 {
		log.Errorf("Invalid amount: %d (must be positive)", amountCreditMicros)
		return fmt.Errorf("invalid amount: %d", amountCreditMicros)
	}

	processed, err := h.checkPaddleIdempotency(ctx, eventID)
	if err != nil {
		return fmt.Errorf("could not check idempotency: %w", err)
	}
	if processed {
		log.Debugf("Event already processed, skipping. event_id: %s", eventID)
		return nil
	}

	// Use existing GetByCustomerID (goes through customer-manager RPC)
	acc, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get account: %v", err)
		return fmt.Errorf("could not get account: %w", err)
	}
	log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	return h.db.AccountPaddleAddCredit(ctx, acc.ID, amountCreditMicros, acc.CustomerID, idempotencyKey)
}

// PaddleSubscriptionCreate sets up a new subscription on the billing account.
// Uses AccountPaddleTopUpTokens for atomic token reset+billing in one transaction.
func (h *accountHandler) PaddleSubscriptionCreate(ctx context.Context, customerID uuid.UUID, planType account.PlanType, paddleSubID string, paddleCustID string, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "PaddleSubscriptionCreate",
		"customer_id":            customerID,
		"plan_type":              planType,
		"paddle_subscription_id": paddleSubID,
		"event_id":               eventID,
	})

	processed, err := h.checkPaddleIdempotency(ctx, eventID)
	if err != nil {
		return fmt.Errorf("could not check idempotency: %w", err)
	}
	if processed {
		log.Debugf("Event already processed, skipping. event_id: %s", eventID)
		return nil
	}

	acc, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("could not get account: %w", err)
	}
	log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

	// Store paddle IDs FIRST — ensures subsequent renewal/cancel events
	// can find this account by paddle_subscription_id even if later steps fail.
	// On Paddle retry, these will be overwritten with the same values (safe). (R4-I1 fix)
	//
	// NOTE: Steps 1-3 (store IDs → update plan → top-up tokens) are not wrapped
	// in a single DB transaction. If step 2 succeeds but step 3 fails, the account
	// will have the new plan type but old token balance until Paddle retries (within minutes).
	// This is an acceptable tradeoff — collapsing into one TX would require restructuring
	// UpdatePlanType, and Paddle retries will complete the operation.
	fields := map[account.Field]any{
		account.FieldPaddleSubscriptionID: paddleSubID,
		account.FieldPaddleCustomerID:     paddleCustID,
	}
	if err := h.db.AccountUpdate(ctx, acc.ID, fields); err != nil {
		return fmt.Errorf("could not update paddle IDs: %w", err)
	}

	// Update plan type
	if _, err := h.UpdatePlanType(ctx, acc.ID, planType); err != nil {
		return fmt.Errorf("could not update plan type: %w", err)
	}

	// Reset tokens to plan allowance — atomic DB method creates billing record
	tokenAllowance, ok := account.PlanTokenMap[planType]
	if !ok {
		return fmt.Errorf("unknown plan type: %s", planType)
	}
	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	return h.db.AccountPaddleTopUpTokens(ctx, acc.ID, acc.CustomerID, tokenAllowance, string(planType), billing.TransactionTypeTopUp, idempotencyKey)
}

// PaddleSubscriptionUpdate changes the plan type when a subscription is upgraded/downgraded.
// Resets tokens to the new plan's allowance (R2-I3 fix).
func (h *accountHandler) PaddleSubscriptionUpdate(ctx context.Context, paddleSubID string, newPlanType account.PlanType, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "PaddleSubscriptionUpdate",
		"paddle_subscription_id": paddleSubID,
		"new_plan_type":          newPlanType,
		"event_id":               eventID,
	})

	processed, err := h.checkPaddleIdempotency(ctx, eventID)
	if err != nil {
		return fmt.Errorf("could not check idempotency: %w", err)
	}
	if processed {
		log.Debugf("Event already processed, skipping. event_id: %s", eventID)
		return nil
	}

	acc, err := h.db.AccountGetByPaddleSubscriptionID(ctx, paddleSubID)
	if err != nil {
		return fmt.Errorf("could not get account by paddle subscription ID: %w", err)
	}
	log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

	if _, err := h.UpdatePlanType(ctx, acc.ID, newPlanType); err != nil {
		return fmt.Errorf("could not update plan type: %w", err)
	}

	// Reset tokens to new plan allowance — atomic DB method creates billing record
	tokenAllowance, ok := account.PlanTokenMap[newPlanType]
	if !ok {
		return fmt.Errorf("unknown plan type: %s", newPlanType)
	}
	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	return h.db.AccountPaddleTopUpTokens(ctx, acc.ID, acc.CustomerID, tokenAllowance, string(newPlanType), billing.TransactionTypeAdjustment, idempotencyKey)
}

// PaddleSubscriptionCancel downgrades the account to Free plan immediately.
// Keeps paddle_subscription_id for post-cancel event correlation (R2-I2 fix).
func (h *accountHandler) PaddleSubscriptionCancel(ctx context.Context, paddleSubID string, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "PaddleSubscriptionCancel",
		"paddle_subscription_id": paddleSubID,
		"event_id":               eventID,
	})

	processed, err := h.checkPaddleIdempotency(ctx, eventID)
	if err != nil {
		return fmt.Errorf("could not check idempotency: %w", err)
	}
	if processed {
		log.Debugf("Event already processed, skipping. event_id: %s", eventID)
		return nil
	}

	acc, err := h.db.AccountGetByPaddleSubscriptionID(ctx, paddleSubID)
	if err != nil {
		return fmt.Errorf("could not get account by paddle subscription ID: %w", err)
	}
	log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

	// Downgrade to free immediately
	if _, err := h.UpdatePlanType(ctx, acc.ID, account.PlanTypeFree); err != nil {
		return fmt.Errorf("could not update plan type: %w", err)
	}

	// NOTE: Do NOT clear paddle_subscription_id — Paddle may still send
	// follow-up events (e.g., transaction.refunded) that need subscription lookup.

	// Reset tokens to free plan allowance — atomic DB method creates billing record
	tokenAllowance, ok := account.PlanTokenMap[account.PlanTypeFree]
	if !ok {
		return fmt.Errorf("unknown plan type: %s", account.PlanTypeFree)
	}
	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	return h.db.AccountPaddleTopUpTokens(ctx, acc.ID, acc.CustomerID, tokenAllowance, string(account.PlanTypeFree), billing.TransactionTypeAdjustment, idempotencyKey)
}

// PaddleSubscriptionRenew replenishes tokens for a subscription renewal.
// Uses AccountPaddleTopUpTokens for atomic token reset+billing in one transaction.
func (h *accountHandler) PaddleSubscriptionRenew(ctx context.Context, paddleSubID string, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "PaddleSubscriptionRenew",
		"paddle_subscription_id": paddleSubID,
		"event_id":               eventID,
	})

	processed, err := h.checkPaddleIdempotency(ctx, eventID)
	if err != nil {
		return fmt.Errorf("could not check idempotency: %w", err)
	}
	if processed {
		log.Debugf("Event already processed, skipping. event_id: %s", eventID)
		return nil
	}

	acc, err := h.db.AccountGetByPaddleSubscriptionID(ctx, paddleSubID)
	if err != nil {
		return fmt.Errorf("could not get account by paddle subscription ID: %w", err)
	}
	log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

	// Guard: skip renewal for cancelled subscriptions (plan downgraded to Free).
	// After PaddleSubscriptionCancel, paddle_subscription_id is kept for event correlation
	// but PlanType is reset to Free. A post-cancel renewal should not grant free-tier tokens. (R4-I2 fix)
	if acc.PlanType == account.PlanTypeFree {
		log.Infof("Skipping renewal for free-plan account (likely post-cancellation). account_id: %s, paddle_subscription_id: %s", acc.ID, paddleSubID)
		return nil
	}

	// Reset tokens to plan allowance — atomic DB method creates billing record
	// For unlimited plans (tokenAllowance=0), still creates audit record with AmountToken=0
	tokenAllowance, ok := account.PlanTokenMap[acc.PlanType]
	if !ok {
		return fmt.Errorf("unknown plan type: %s", acc.PlanType)
	}
	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	return h.db.AccountPaddleTopUpTokens(ctx, acc.ID, acc.CustomerID, tokenAllowance, string(acc.PlanType), billing.TransactionTypeTopUp, idempotencyKey)
}

// PaddleRefund subtracts credit from a Paddle refund.
// Uses AccountPaddleSubtractCredit for atomic balance subtract+billing in one transaction.
//
// NOTE: Uses details.totals.total from the Paddle event. Verify against Paddle v2 docs
// whether this represents the refund amount or the original transaction total — for partial
// refunds the correct field may differ. Must be verified before production deployment.
func (h *accountHandler) PaddleRefund(ctx context.Context, customerID uuid.UUID, amountCreditMicros int64, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PaddleRefund",
		"customer_id": customerID,
		"amount":      amountCreditMicros,
		"event_id":    eventID,
	})

	if amountCreditMicros <= 0 {
		log.Errorf("Invalid refund amount: %d (must be positive)", amountCreditMicros)
		return fmt.Errorf("invalid refund amount: %d", amountCreditMicros)
	}

	processed, err := h.checkPaddleIdempotency(ctx, eventID)
	if err != nil {
		return fmt.Errorf("could not check idempotency: %w", err)
	}
	if processed {
		log.Debugf("Event already processed, skipping. event_id: %s", eventID)
		return nil
	}

	acc, err := h.GetByCustomerID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("could not get account: %w", err)
	}
	log.WithField("account", acc).Debugf("Retrieved account info. account_id: %s", acc.ID)

	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)
	if err := h.db.AccountPaddleSubtractCredit(ctx, acc.ID, amountCreditMicros, acc.CustomerID, idempotencyKey); err != nil {
		return fmt.Errorf("could not subtract balance: %w", err)
	}

	// Check if balance went negative → freeze
	updatedAcc, err := h.db.AccountGet(ctx, acc.ID)
	if err != nil {
		return fmt.Errorf("could not get updated account: %w", err)
	}
	if updatedAcc.BalanceCredit < 0 {
		log.Infof("Account balance negative after refund, freezing. account_id: %s, balance: %d", acc.ID, updatedAcc.BalanceCredit)
		if _, err := h.SetStatus(ctx, acc.ID, account.StatusFrozen); err != nil {
			log.Errorf("Could not freeze account: %v", err)
		}
	}

	return nil
}
