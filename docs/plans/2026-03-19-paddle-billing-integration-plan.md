# Paddle Billing Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate Paddle Billing v2 to enable credit top-ups, subscription management, and refund handling via Paddle webhooks.

**Architecture:** bin-hook-manager receives Paddle webhooks at `POST /v1.0/billing/paddle`, verifies the Paddle signature, wraps the payload in a `Hook` struct, and sends it via RabbitMQ RPC to bin-billing-manager. billing-manager parses the Paddle event, checks idempotency, and routes to the appropriate account operation (credit top-up, subscription create/update/cancel, renewal, or refund).

**Tech Stack:** Go 1.21+, Paddle Go SDK v4 (`github.com/PaddleHQ/paddle-go-sdk/v4`), RabbitMQ RPC, MySQL (Alembic migrations), gomock for testing.

**Design doc:** `docs/plans/2026-03-19-paddle-billing-integration-design.md`

**Review resolutions applied:** C1 (use `GetByCustomerID` RPC chain), C2/C3 (add `BillingGetByIdempotencyKey`), C4 (explicit `main.go` update), C5 (use cobra `PersistentFlags`), C6 (use `bin-manager-secrets`), C7 (parse decimal amounts), C8 (immediate downgrade), I1 (explicit `CostTypeNone`), I4 (use `AccountTopUpTokens`), I5 (fixed route), I8 (audit record for unlimited), I9 (use `ApplyFields` pattern), I10 (30-service verification), I12 (log+200 for missing custom_data).

---

## Task 1: Database Migration — Add Paddle columns to billing_accounts

Add `paddle_subscription_id` and `paddle_customer_id` columns with indexes.

**Files:**
- Create: `bin-dbscheme-manager/bin-manager/main/versions/g1a2b3c4d5e6_billing_accounts_add_column_paddle_subscription_id_paddle_customer_id.py`

**Step 1: Create the migration file**

> **NOTE:** The `revision` and `down_revision` values below are placeholders. Before committing, verify the actual current head revision by inspecting the latest migration file in `bin-dbscheme-manager/bin-manager/main/versions/`. Set `down_revision` to the actual latest revision ID.

```python
"""billing_accounts_add_column_paddle_subscription_id_paddle_customer_id

Revision ID: g1a2b3c4d5e6
Revises: f1a2b3c4d5e6
Create Date: 2026-03-19 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'g1a2b3c4d5e6'
down_revision = 'f1a2b3c4d5e6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE billing_accounts
        ADD COLUMN paddle_subscription_id VARCHAR(255) DEFAULT NULL,
        ADD COLUMN paddle_customer_id VARCHAR(255) DEFAULT NULL;
    """)

    op.execute("""
        CREATE INDEX ix_billing_accounts_paddle_subscription_id
        ON billing_accounts(paddle_subscription_id);
    """)

    op.execute("""
        CREATE INDEX ix_billing_accounts_paddle_customer_id
        ON billing_accounts(paddle_customer_id);
    """)


def downgrade():
    op.execute("""
        DROP INDEX ix_billing_accounts_paddle_customer_id ON billing_accounts;
    """)

    op.execute("""
        DROP INDEX ix_billing_accounts_paddle_subscription_id ON billing_accounts;
    """)

    op.execute("""
        ALTER TABLE billing_accounts
        DROP COLUMN paddle_customer_id,
        DROP COLUMN paddle_subscription_id;
    """)
```

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-dbscheme-manager/bin-manager/main/versions/g1a2b3c4d5e6_billing_accounts_add_column_paddle_subscription_id_paddle_customer_id.py
git commit -m "NOJIRA-Paddle-billing-integration

- bin-dbscheme-manager: Add paddle_subscription_id and paddle_customer_id columns to billing_accounts"
```

---

## Task 2: billing-manager Models — Add Paddle fields and ReferenceTypes

Add Paddle fields to Account, Field constants, and new ReferenceType constants.

**Files:**
- Modify: `bin-billing-manager/models/account/account.go`
- Modify: `bin-billing-manager/models/account/field.go`
- Modify: `bin-billing-manager/models/billing/billing.go`

**Step 1: Add fields to Account struct**

In `bin-billing-manager/models/account/account.go`, add after `PaymentMethod PaymentMethod`:

```go
PaddleSubscriptionID string `json:"paddle_subscription_id" db:"paddle_subscription_id"`
PaddleCustomerID     string `json:"paddle_customer_id" db:"paddle_customer_id"`
```

**Step 2: Add Field constants**

In `bin-billing-manager/models/account/field.go`, add:

```go
FieldPaddleSubscriptionID Field = "paddle_subscription_id"
FieldPaddleCustomerID     Field = "paddle_customer_id"
```

**Step 3: Add ReferenceType constants**

In `bin-billing-manager/models/billing/billing.go`, add to the ReferenceType constants block:

```go
ReferenceTypePaddleCreditPurchase ReferenceType = "paddle_credit_purchase"
ReferenceTypePaddleSubscription   ReferenceType = "paddle_subscription"
ReferenceTypePaddleRefund         ReferenceType = "paddle_refund"
```

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-billing-manager/models/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-billing-manager: Add Paddle fields to Account model and Paddle ReferenceType constants"
```

---

## Task 3: billing-manager DBHandler — Add Paddle DB methods

Add `AccountGetByPaddleSubscriptionID` and `BillingGetByIdempotencyKey` to DBHandler.

**Files:**
- Modify: `bin-billing-manager/pkg/dbhandler/main.go` (interface)
- Create: `bin-billing-manager/pkg/dbhandler/account_paddle.go`
- Create: `bin-billing-manager/pkg/dbhandler/billing_paddle.go`

**Step 1: Add to DBHandler interface**

In `bin-billing-manager/pkg/dbhandler/main.go`, add to the `DBHandler` interface:

```go
AccountGetByPaddleSubscriptionID(ctx context.Context, paddleSubscriptionID string) (*account.Account, error)
BillingGetByIdempotencyKey(ctx context.Context, idempotencyKey uuid.UUID) (*billing.Billing, error)
```

**Step 2: Write AccountGetByPaddleSubscriptionID**

Create `bin-billing-manager/pkg/dbhandler/account_paddle.go`:

```go
package dbhandler

import (
	"context"
	"fmt"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	sq "github.com/Masterminds/squirrel"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
)

// AccountGetByPaddleSubscriptionID returns the account matching the given Paddle subscription ID.
func (h *handler) AccountGetByPaddleSubscriptionID(ctx context.Context, paddleSubscriptionID string) (*account.Account, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "AccountGetByPaddleSubscriptionID",
		"paddle_subscription_id": paddleSubscriptionID,
	})

	// Use the same filter pattern as AccountListByCustomerID for soft-delete
	filters := map[account.Field]any{
		account.FieldPaddleSubscriptionID: paddleSubscriptionID,
		account.FieldDeleted:              false,
	}

	accounts, err := h.AccountList(ctx, 1, "", filters)
	if err != nil {
		log.Errorf("Could not list accounts by paddle subscription ID: %v", err)
		return nil, fmt.Errorf("could not query account: %w", err)
	}

	if len(accounts) == 0 {
		return nil, ErrNotFound
	}

	log.WithField("account", accounts[0]).Debugf("Retrieved account by paddle_subscription_id. account_id: %s", accounts[0].ID)
	return accounts[0], nil
}
```

**Step 3: Write BillingGetByIdempotencyKey**

Create `bin-billing-manager/pkg/dbhandler/billing_paddle.go`:

```go
package dbhandler

import (
	"context"
	"fmt"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/billing"
)

// BillingGetByIdempotencyKey returns the billing record with the given idempotency key.
func (h *handler) BillingGetByIdempotencyKey(ctx context.Context, idempotencyKey uuid.UUID) (*billing.Billing, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "BillingGetByIdempotencyKey",
		"idempotency_key": idempotencyKey,
	})

	var res billing.Billing
	cols := commondatabasehandler.GetDBFields(&res)

	query, args, err := sq.Select(cols...).
		From(billingsTable).
		Where(sq.Eq{"idempotency_key": idempotencyKey.Bytes()}).
		Where(sq.Expr("tm_delete IS NULL")).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	if err := commondatabasehandler.ScanRow(rows, &res); err != nil {
		log.Errorf("Could not scan row: %v", err)
		return nil, fmt.Errorf("could not scan row: %w", err)
	}

	return &res, nil
}
```

> **NOTE:** Check `billingsTable` constant name — look at existing `billing.go` in dbhandler for the correct table name constant. The soft-delete pattern for billings uses `WHERE tm_delete IS NULL` (confirmed from existing `billing.go:154`).

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-billing-manager/pkg/dbhandler/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-billing-manager: Add AccountGetByPaddleSubscriptionID and BillingGetByIdempotencyKey to DBHandler"
```

---

## Task 4: billing-manager AccountHandler — Add Paddle methods

Add business logic methods for all Paddle operations.

**Files:**
- Modify: `bin-billing-manager/pkg/accounthandler/main.go` (interface)
- Create: `bin-billing-manager/pkg/accounthandler/paddle.go`
- Create: `bin-billing-manager/pkg/accounthandler/paddle_test.go`

**Step 1: Add interface methods**

In `bin-billing-manager/pkg/accounthandler/main.go`, add to `AccountHandler` interface:

```go
// Paddle webhook handlers
PaddleCreditTopUp(ctx context.Context, customerID uuid.UUID, amountCreditMicros int64, eventID string) error
PaddleSubscriptionCreate(ctx context.Context, customerID uuid.UUID, planType account.PlanType, paddleSubID string, paddleCustID string, eventID string) error
PaddleSubscriptionUpdate(ctx context.Context, paddleSubID string, newPlanType account.PlanType, eventID string) error
PaddleSubscriptionCancel(ctx context.Context, paddleSubID string, eventID string) error
PaddleSubscriptionRenew(ctx context.Context, paddleSubID string, eventID string) error
PaddleRefund(ctx context.Context, customerID uuid.UUID, amountCreditMicros int64, eventID string) error
```

> **NOTE:** `PaddleSubscriptionCancel` no longer takes `effectiveFrom` — it always downgrades immediately (C8 fix). Paddle fires the event at end of billing period when user chose end-of-period cancellation.

**Step 2: Write implementation**

Create `bin-billing-manager/pkg/accounthandler/paddle.go`. Key patterns to follow:

- **Customer lookup** uses existing `h.GetByCustomerID(ctx, customerID)` which goes through customer-manager RPC (C1 fix)
- **Idempotency** uses single `h.db.BillingGetByIdempotencyKey(ctx, key)` query (C2/C3 fix)
- **Subscription token allocation** uses `h.db.AccountTopUpTokens()` (resets tokens, not additive) (I4 fix)
- **CostType** always set to `billing.CostTypeNone` explicitly (I1 fix)
- **Unlimited plan renewals** still create a billing record with `AmountToken: 0` for audit trail (I8 fix)

```go
package accounthandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/dbhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
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

// createPaddleBillingRecord creates an immutable billing record for a Paddle event.
func (h *accountHandler) createPaddleBillingRecord(ctx context.Context, acc *account.Account, txnType billing.TransactionType, refType billing.ReferenceType, amountCredit int64, amountToken int64, eventID string) error {
	idempotencyKey := uuid.NewV5(uuid.NamespaceDNS, eventID)

	// Get latest account state for snapshot
	updatedAcc, err := h.db.AccountGet(ctx, acc.ID)
	if err != nil {
		return fmt.Errorf("could not get updated account for snapshot: %w", err)
	}

	bill := &billing.Billing{
		Identity: commonidentity.Identity{
			ID:         h.utilHandler.UUIDCreate(),
			CustomerID: acc.CustomerID,
		},
		AccountID:             acc.ID,
		TransactionType:       txnType,
		Status:                billing.StatusFinished,
		ReferenceType:         refType,
		ReferenceID:           idempotencyKey,
		CostType:              billing.CostTypeNone,
		AmountCredit:          amountCredit,
		AmountToken:           amountToken,
		BalanceCreditSnapshot: updatedAcc.BalanceCredit,
		BalanceTokenSnapshot:  updatedAcc.BalanceToken,
		IdempotencyKey:        idempotencyKey,
	}

	if err := h.db.BillingCreate(ctx, bill); err != nil {
		return fmt.Errorf("could not create billing record: %w", err)
	}
	return nil
}

// PaddleCreditTopUp adds credit balance from a Paddle credit purchase.
func (h *accountHandler) PaddleCreditTopUp(ctx context.Context, customerID uuid.UUID, amountCreditMicros int64, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PaddleCreditTopUp",
		"customer_id": customerID,
		"amount":      amountCreditMicros,
		"event_id":    eventID,
	})

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

	if err := h.db.AccountAddBalance(ctx, acc.ID, amountCreditMicros); err != nil {
		return fmt.Errorf("could not add balance: %w", err)
	}

	return h.createPaddleBillingRecord(ctx, acc, billing.TransactionTypeTopUp, billing.ReferenceTypePaddleCreditPurchase, amountCreditMicros, 0, eventID)
}

// PaddleSubscriptionCreate sets up a new subscription on the billing account.
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

	// Update plan type
	if _, err := h.UpdatePlanType(ctx, acc.ID, planType); err != nil {
		return fmt.Errorf("could not update plan type: %w", err)
	}

	// Store paddle IDs
	fields := map[account.Field]any{
		account.FieldPaddleSubscriptionID: paddleSubID,
		account.FieldPaddleCustomerID:     paddleCustID,
	}
	if err := h.db.AccountUpdate(ctx, acc.ID, fields); err != nil {
		return fmt.Errorf("could not update paddle IDs: %w", err)
	}

	// Reset tokens to plan allowance (not additive — uses AccountTopUpTokens)
	tokenAllowance, ok := account.PlanTokenMap[planType]
	if ok && tokenAllowance > 0 {
		if err := h.db.AccountTopUpTokens(ctx, acc.ID, acc.CustomerID, tokenAllowance, string(planType)); err != nil {
			return fmt.Errorf("could not top up tokens: %w", err)
		}
	}

	return h.createPaddleBillingRecord(ctx, acc, billing.TransactionTypeTopUp, billing.ReferenceTypePaddleSubscription, 0, tokenAllowance, eventID)
}

// PaddleSubscriptionUpdate changes the plan type when a subscription is upgraded/downgraded.
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

	return h.createPaddleBillingRecord(ctx, acc, billing.TransactionTypeAdjustment, billing.ReferenceTypePaddleSubscription, 0, 0, eventID)
}

// PaddleSubscriptionCancel downgrades the account to Free plan immediately.
// Paddle fires subscription.canceled at end of billing period when user chose end-of-period cancellation.
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

	// Clear paddle subscription ID
	fields := map[account.Field]any{
		account.FieldPaddleSubscriptionID: "",
	}
	if err := h.db.AccountUpdate(ctx, acc.ID, fields); err != nil {
		return fmt.Errorf("could not clear paddle subscription ID: %w", err)
	}

	return h.createPaddleBillingRecord(ctx, acc, billing.TransactionTypeAdjustment, billing.ReferenceTypePaddleSubscription, 0, 0, eventID)
}

// PaddleSubscriptionRenew replenishes tokens for a subscription renewal.
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

	tokenAllowance, ok := account.PlanTokenMap[acc.PlanType]
	if ok && tokenAllowance > 0 {
		if err := h.db.AccountTopUpTokens(ctx, acc.ID, acc.CustomerID, tokenAllowance, string(acc.PlanType)); err != nil {
			return fmt.Errorf("could not top up tokens: %w", err)
		}
	}

	// Always create billing record for audit trail (even for unlimited plans with 0 tokens)
	return h.createPaddleBillingRecord(ctx, acc, billing.TransactionTypeTopUp, billing.ReferenceTypePaddleSubscription, 0, tokenAllowance, eventID)
}

// PaddleRefund subtracts credit from a Paddle refund.
func (h *accountHandler) PaddleRefund(ctx context.Context, customerID uuid.UUID, amountCreditMicros int64, eventID string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "PaddleRefund",
		"customer_id": customerID,
		"amount":      amountCreditMicros,
		"event_id":    eventID,
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

	// Subtract balance (allow negative)
	if err := h.db.AccountSubtractBalance(ctx, acc.ID, amountCreditMicros); err != nil {
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

	// Negative delta for refund
	return h.createPaddleBillingRecord(ctx, acc, billing.TransactionTypeRefund, billing.ReferenceTypePaddleRefund, -amountCreditMicros, 0, eventID)
}
```

**Step 3: Write tests**

Create `bin-billing-manager/pkg/accounthandler/paddle_test.go` with table-driven tests for each method. Use gomock for `dbhandler.MockDBHandler` and `requesthandler.MockRequestHandler`. Key mock expectations:

- `PaddleCreditTopUp`: mock `BillingGetByIdempotencyKey` (returns `ErrNotFound`), mock `reqHandler.CustomerV1CustomerGet` + `db.AccountGet` (via `GetByCustomerID`), mock `AccountAddBalance`, mock `AccountGet` (snapshot), mock `BillingCreate`
- `PaddleSubscriptionCreate`: similar, plus `AccountUpdate` (paddle IDs) and `AccountTopUpTokens`
- Idempotency: test where `BillingGetByIdempotencyKey` returns a record — method should return nil without side effects

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-billing-manager/pkg/accounthandler/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-billing-manager: Add Paddle webhook handler methods to AccountHandler"
```

---

## Task 5: billing-manager ListenHandler — Add Paddle webhook route

**Files:**
- Modify: `bin-billing-manager/pkg/listenhandler/main.go` (regex + switch case)
- Create: `bin-billing-manager/pkg/listenhandler/v1_hooks_paddle.go`
- Create: `bin-billing-manager/pkg/listenhandler/v1_hooks_paddle_test.go`

**Step 1: Add regex and switch case**

In `bin-billing-manager/pkg/listenhandler/main.go`, add regex:

```go
regV1HooksPaddle = regexp.MustCompile("/v1/hooks/paddle$")
```

Add to the `processRequest` switch statement (before the default case):

```go
// POST /hooks/paddle
case regV1HooksPaddle.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1HooksPaddlePost(ctx, m)
    requestType = "/v1/hooks/paddle"
```

**Step 2: Write the listen handler**

Create `bin-billing-manager/pkg/listenhandler/v1_hooks_paddle.go`. Key implementation details:

- Parse Paddle event JSON with minimal struct types (not full SDK types — billing-manager doesn't import Paddle SDK)
- **Amount parsing**: Use `strconv.ParseFloat` then multiply by 1,000,000 for micros (C7 fix)
- **Missing custom_data**: Log warning and return 200 (I12 fix — prevents retry storm)
- Route to accountHandler methods based on `event_type`

```go
// parsePaddleAmountToMicros converts a Paddle decimal amount string to micros.
// Paddle v2 sends amounts as decimal strings: "10.00" = $10.00 = 10,000,000 micros.
func parsePaddleAmountToMicros(amountStr string) (int64, error) {
    amount, err := strconv.ParseFloat(amountStr, 64)
    if err != nil {
        return 0, fmt.Errorf("could not parse amount %q: %w", amountStr, err)
    }
    return int64(math.Round(amount * 1_000_000)), nil
}
```

For `subscription.created` and `transaction.completed` (one-time): if `custom_data` is missing, log and return 200:

```go
if txnData.CustomData == nil || txnData.CustomData.CustomerID == "" {
    log.Infof("Missing customer_id in custom_data, skipping. event_id: %s", event.EventID)
    return simpleResponse(200), nil
}
```

**Step 3: Write tests**

Create `bin-billing-manager/pkg/listenhandler/v1_hooks_paddle_test.go` covering:
- Transaction completed (one-time credit purchase)
- Transaction completed (subscription renewal — has subscription_id)
- Subscription created/updated/canceled
- Transaction refunded
- Unknown event type → 200
- Missing custom_data → 200 (not 400)
- `parsePaddleAmountToMicros` unit tests: `"10.00"` → `10000000`, `"0.50"` → `500000`

**Step 4: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-billing-manager/pkg/listenhandler/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-billing-manager: Add Paddle webhook listen handler at /v1/hooks/paddle"
```

---

## Task 6: bin-common-handler — Add BillingV1PaddleHook to RequestHandler

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` (interface)
- Create: `bin-common-handler/pkg/requesthandler/billing_hooks.go`

**Step 1: Add to RequestHandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, add near the existing billing methods:

```go
BillingV1PaddleHook(ctx context.Context, hm *hmhook.Hook) error
```

**Step 2: Write implementation**

Create `bin-common-handler/pkg/requesthandler/billing_hooks.go` following the `email_hooks.go` pattern exactly:

```go
package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	hmhook "monorepo/bin-hook-manager/models/hook"
)

// BillingV1PaddleHook sends a Paddle webhook hook to billing-manager
func (r *requestHandler) BillingV1PaddleHook(ctx context.Context, hm *hmhook.Hook) error {
	uri := "/v1/hooks/paddle"

	m, err := json.Marshal(hm)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/hooks/paddle", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
```

**Step 3: Run verification for bin-common-handler**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Regenerate mocks in ALL consumer services**

Adding a method to the `RequestHandler` interface requires mock regeneration in every service that imports it. Run `go generate ./...` and `go test ./...` in at minimum: `bin-billing-manager`, `bin-hook-manager`, and any other service whose tests mock `RequestHandler`.

> **NOTE (I10):** The safest approach is to run verification in ALL 30+ services. At minimum, run `go generate ./...` in each service that imports `bin-common-handler/pkg/requesthandler`. Compilation errors will surface in `go test` if any mock is stale.

**Step 5: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-common-handler/pkg/requesthandler/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-common-handler: Add BillingV1PaddleHook RPC method to RequestHandler"
```

---

## Task 7: hook-manager — Refactor ServiceHandler interface to use *http.Request

**Files:**
- Modify: `bin-hook-manager/pkg/servicehandler/main.go`
- Modify: `bin-hook-manager/pkg/servicehandler/email.go`
- Modify: `bin-hook-manager/pkg/servicehandler/message.go`
- Modify: `bin-hook-manager/pkg/servicehandler/conversation.go`
- Modify: `bin-hook-manager/api/v1.0/emails/emails.go`
- Modify: `bin-hook-manager/api/v1.0/messages/messages.go`
- Modify: `bin-hook-manager/api/v1.0/conversation/conversation.go`
- Modify: All `*_test.go` files for these packages

**Step 1: Update interface**

In `bin-hook-manager/pkg/servicehandler/main.go`:

```go
import "net/http"

type ServiceHandler interface {
    Email(ctx context.Context, r *http.Request) error
    Message(ctx context.Context, r *http.Request) error
    Conversation(ctx context.Context, r *http.Request) error
    Billing(ctx context.Context, r *http.Request) error  // Added next task
}
```

**Step 2: Update each handler to read body internally**

Each handler (`email.go`, `message.go`, `conversation.go`) reads body from `r.Body` and constructs the `Hook` struct internally. Example for `email.go`:

```go
func (h *serviceHandler) Email(ctx context.Context, r *http.Request) error {
    data, err := io.ReadAll(r.Body)
    if err != nil {
        return errors.Wrap(err, "could not read body")
    }

    req := &hmhook.Hook{
        ReceviedURI:  r.Host + r.URL.Path,
        ReceivedData: data,
    }

    if errHook := h.reqHandler.EmailV1Hooks(ctx, req); errHook != nil {
        return errors.Wrapf(errHook, "could not send the hook")
    }
    return nil
}
```

**Step 3: Simplify Gin handlers**

Remove `io.ReadAll` from Gin handlers. Pass `c.Request` directly:

```go
func emailsPOST(c *gin.Context) {
    ctx := context.Background()
    serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
    if errHandler := serviceHandler.Email(ctx, c.Request); errHandler != nil {
        c.AbortWithStatus(http.StatusInternalServerError)
        return
    }
    c.AbortWithStatus(200)
}
```

**Step 4: Update all tests**

Update all test files to construct `*http.Request` via `httptest.NewRequest("POST", "/...", bytes.NewReader(body))` instead of passing `(uri, data)`.

**Step 5: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-hook-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 6: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-hook-manager/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-hook-manager: Refactor ServiceHandler interface to pass *http.Request instead of (uri, data)"
```

---

## Task 8: hook-manager — Add Billing handler with Paddle signature verification

**Files:**
- Create: `bin-hook-manager/api/v1.0/billing/main.go`
- Create: `bin-hook-manager/api/v1.0/billing/billing.go`
- Create: `bin-hook-manager/pkg/servicehandler/billing.go`
- Modify: `bin-hook-manager/api/v1.0/v1.0.go` (add billing routes)
- Modify: `bin-hook-manager/pkg/servicehandler/main.go` (add paddleVerifier field)
- Modify: `bin-hook-manager/cmd/hook-manager/main.go` (pass paddle secret)
- Modify: `bin-hook-manager/internal/config/config.go` (add config field)
- Modify: `bin-hook-manager/go.mod` (add Paddle SDK)
- Modify: `bin-hook-manager/k8s/deployment.yml` (add env var)

**Step 1: Add Paddle SDK dependency**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-hook-manager
go get github.com/PaddleHQ/paddle-go-sdk/v4
```

**Step 2: Add config for Paddle webhook secret (C5 fix)**

In `bin-hook-manager/internal/config/config.go`:

Add to `Config` struct:
```go
PaddleWebhookSecretKey string
```

Add to `bindConfig` function's `f` declarations:
```go
f.String("paddle_webhook_secret_key", "", "Paddle webhook secret key for signature verification")
```

Add to `bindConfig` function's `bindings` map:
```go
"paddle_webhook_secret_key": "PADDLE_WEBHOOK_SECRET_KEY",
```

Add to `LoadGlobalConfig`'s `cfg` initialization:
```go
PaddleWebhookSecretKey: viper.GetString("paddle_webhook_secret_key"),
```

Also add to the legacy `InitConfig` function:
```go
_ = viper.BindPFlag("paddle_webhook_secret_key", cmd.Flags().Lookup("paddle_webhook_secret_key"))
```
And add the field to the `cfg` struct in `InitConfig`.

> **NOTE:** `main.go` uses `rootCmd.Flags()` for flag definitions and `InitConfig` for viper binding. Add the flag in `init()`:
```go
rootCmd.Flags().String("paddle_webhook_secret_key", "", "Paddle webhook secret key")
```

**Step 3: Update serviceHandler struct and NewServiceHandler (C4 fix)**

In `bin-hook-manager/pkg/servicehandler/main.go`, update:

```go
type serviceHandler struct {
    reqHandler     requesthandler.RequestHandler
    paddleVerifier *paddle.WebhookVerifier
}

func NewServiceHandler(reqHandler requesthandler.RequestHandler, paddleWebhookSecret string) ServiceHandler {
    var verifier *paddle.WebhookVerifier
    if paddleWebhookSecret != "" {
        v, err := paddle.NewWebhookVerifier(paddleWebhookSecret)
        if err == nil {
            verifier = v
        }
    }
    return &serviceHandler{
        reqHandler:     reqHandler,
        paddleVerifier: verifier,
    }
}
```

> **NOTE:** Verify the Paddle SDK constructor name during implementation. Check if it's `paddle.NewWebhookVerifier` or a different API.

**Step 4: Update main.go to pass the paddle secret (C4 fix)**

In `bin-hook-manager/cmd/hook-manager/main.go`, change line 112:

```go
// OLD:
serviceHandler := servicehandler.NewServiceHandler(requestHandler)

// NEW:
serviceHandler := servicehandler.NewServiceHandler(requestHandler, cfg.PaddleWebhookSecretKey)
```

**Step 5: Write billing service handler**

Create `bin-hook-manager/pkg/servicehandler/billing.go` — reads body, restores for verification, sends RPC. On signature failure, return error (Gin handler converts to 400 status for signature failures).

**Step 6: Create billing API route (I5 fix — fixed route, not wildcard)**

Create `bin-hook-manager/api/v1.0/billing/main.go`:

```go
package billing

import "github.com/gin-gonic/gin"

func ApplyRoutes(r *gin.RouterGroup) {
    g := r.Group("/billing")
    g.POST("/paddle", billingPaddlePOST)  // Fixed route, not /:target
}
```

**Step 7: Register billing routes**

In `bin-hook-manager/api/v1.0/v1.0.go`:

```go
import "monorepo/bin-hook-manager/api/v1.0/billing"
// In ApplyRoutes:
billing.ApplyRoutes(v1)
```

**Step 8: Add K8s env var (C6 fix — correct secret name)**

In `bin-hook-manager/k8s/deployment.yml`, add to env section:

```yaml
- name: PADDLE_WEBHOOK_SECRET_KEY
  valueFrom:
    secretKeyRef:
      name: bin-manager-secrets
      key: PADDLE_WEBHOOK_SECRET_KEY
```

**Step 9: Write tests and run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-hook-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 10: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git add bin-hook-manager/
git commit -m "NOJIRA-Paddle-billing-integration

- bin-hook-manager: Add billing webhook endpoint at POST /v1.0/billing/paddle with Paddle signature verification
- bin-hook-manager: Add Paddle Go SDK v4 dependency
- bin-hook-manager: Add PADDLE_WEBHOOK_SECRET_KEY config and K8s secret reference"
```

---

## Task 9: Cross-service verification

Run full verification for all changed services to catch interface mismatches.

**Step 1: Verify bin-common-handler**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-billing-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-billing-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-hook-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/bin-hook-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Spot-check other services that import requesthandler**

At minimum, verify a few other services compile with the new RequestHandler interface:

```bash
for svc in bin-api-manager bin-call-manager bin-customer-manager; do
    cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration/$svc
    go mod tidy && go mod vendor && go generate ./... && go build ./...
done
```

**Step 5: Fix any issues found and re-verify**

---

## Task 10: Create PR

**Step 1: Fetch latest main and check for conflicts**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Paddle-billing-integration
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 2: Push and create PR**

```bash
git push -u origin NOJIRA-Paddle-billing-integration
gh pr create --title "NOJIRA-Paddle-billing-integration" --body "$(cat <<'EOF'
Integrate Paddle Billing v2 to enable credit top-ups, subscription management,
and refund handling via Paddle webhooks.

- bin-dbscheme-manager: Add paddle_subscription_id and paddle_customer_id columns to billing_accounts
- bin-billing-manager: Add PaddleSubscriptionID and PaddleCustomerID to Account model
- bin-billing-manager: Add Paddle ReferenceType constants (paddle_credit_purchase, paddle_subscription, paddle_refund)
- bin-billing-manager: Add AccountGetByPaddleSubscriptionID and BillingGetByIdempotencyKey to DBHandler
- bin-billing-manager: Add Paddle webhook handler methods to AccountHandler (credit top-up, subscription CRUD, refund)
- bin-billing-manager: Add Paddle webhook listen handler at /v1/hooks/paddle
- bin-common-handler: Add BillingV1PaddleHook RPC method to RequestHandler
- bin-hook-manager: Refactor ServiceHandler interface to pass *http.Request instead of (uri, data)
- bin-hook-manager: Add billing webhook endpoint at POST /v1.0/billing/paddle with Paddle signature verification
- bin-hook-manager: Add Paddle Go SDK v4 dependency
- bin-hook-manager: Add PADDLE_WEBHOOK_SECRET_KEY config and K8s secret reference
EOF
)"
```
