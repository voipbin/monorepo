# Free Tier Monthly Credit Top-Up

## Problem Statement

Free-tier users start with a $0.00 balance, which means they cannot use any paid services (calls, SMS, number purchases). To encourage trial usage, we want to automatically provide $1.00/month in free credits to every free-tier billing account.

## Approach

A daily ticker in `bin-billing-manager` checks all free-tier billing accounts and tops up eligible accounts to $1.00. Each top-up creates a billing record for audit trail and concurrency safety.

### Top-Up Logic

For each free-tier account, the daily check:

1. Generate a deterministic `reference_id` for the current month using UUID v5.
2. Build a billing record struct with the deterministic `reference_id` and `cost_total = 0` (placeholder).
3. Call `BillingCreditTopUp` which atomically:
   a. Inserts the billing record (duplicate key check via unique index).
   b. If duplicate — already processed this month, rollback and return `(false, nil)`.
   c. Locks the account row and reads current balance.
   d. If balance >= $1.00 — no balance change, commit with `cost_total = 0` to prevent re-processing.
   e. If balance < $1.00 — calculate delta ($1.00 - balance), update balance and `cost_total`, commit.
4. If any step fails — transaction rolls back, no partial state. Next daily run retries.

### Example Scenarios

| Current Balance | Last Top-Up | Action | New Balance |
|----------------|-------------|--------|-------------|
| $0.00 | Never | Add $1.00 | $1.00 |
| $0.90 | 35 days ago | Add $0.10 | $1.00 |
| $1.00 | 35 days ago | Insert record with cost_total=0, skip balance add | $1.00 |
| $1.50 | 35 days ago | Insert record with cost_total=0, skip balance add | $1.50 |
| $0.50 | 15 days ago | Duplicate key, skip entirely | $0.50 |

Note: Records with `cost_total = 0` still serve a purpose — they prevent the system from re-checking this account every day for the rest of the month. Without the record, the daily ticker would query the balance, find it >= $1.00, and do nothing — every single day. With the record, the duplicate key check skips it instantly.

### Concurrency Safety (Multiple Pods)

In a Kubernetes environment, multiple pods may run the daily ticker simultaneously. We handle this without expensive row locking by using **deterministic UUID v5 reference IDs**.

The billing table has a unique index on `(reference_type, reference_id)`. By generating the same `reference_id` for a given account + month across all pods, only the first pod's insert succeeds — subsequent pods hit a duplicate key error and skip.

A `reference_id` should never be duplicated within a `reference_type` — each billing record represents a unique billable event. This index enforces that invariant for all billing record types, not just credits.

```go
// We use uuid.Nil as the namespace for UUID v5 generation.
// uuid.Nil is the zero-value UUID (00000000-0000-0000-0000-000000000000).
// This is intentional — we don't need a custom namespace because the input data
// (accountID + year-month) is already globally unique. The deterministic UUID
// ensures that all pods generate the same reference_id for the same account
// and month, which leverages the unique index on billing_billings
// (reference_type, reference_id) to prevent duplicate top-ups
// without expensive row-level locking.
referenceID := h.utilHandler.NewV5UUID(uuid.Nil, accountID.String()+":"+currentYearMonth)
```

### Atomicity via Transaction

The billing record insert and balance update are wrapped in a single DB transaction to prevent partial state and ensure accurate balance calculation. The balance is read inside the transaction with `FOR UPDATE` to prevent concurrent modifications from producing incorrect deltas.

New DB method `BillingCreditTopUp(ctx, billing, accountID, targetAmount) (created bool, err error)`:

```go
func (h *handler) BillingCreditTopUp(ctx context.Context, b *billing.Billing, accountID uuid.UUID, targetAmount float32) (bool, error) {
    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil {
        return false, err
    }
    defer tx.Rollback()

    // 1. Insert billing record (tm_delete = nil, not sentinel)
    b.TMCreate = h.utilHandler.TimeNow()
    b.TMUpdate = nil
    b.TMDelete = nil
    fields, err := commondatabasehandler.PrepareFields(b)
    if err != nil {
        return false, fmt.Errorf("could not prepare fields. err: %v", err)
    }
    query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
    if err != nil {
        return false, fmt.Errorf("could not build query. err: %v", err)
    }
    _, err = tx.ExecContext(ctx, query, args...)
    if err != nil {
        if isDuplicateKeyError(err) {
            return false, nil // already processed this month
        }
        return false, fmt.Errorf("could not insert billing. err: %v", err)
    }

    // 2. Lock account row and read current balance
    var balance float32
    row := tx.QueryRowContext(ctx, "SELECT balance FROM billing_accounts WHERE id = ? FOR UPDATE", accountID.Bytes())
    if err := row.Scan(&balance); err != nil {
        return false, fmt.Errorf("could not read balance. err: %v", err)
    }

    // 3. Calculate delta inside transaction (accurate, no race condition).
    // targetAmount is passed from the caller to avoid circular import
    // (credithandler defines the constant, dbhandler must not import credithandler).
    delta := targetAmount - balance
    if delta > 0 {
        now := h.utilHandler.TimeNow()
        _, err = tx.ExecContext(ctx,
            "UPDATE billing_accounts SET balance = balance + ?, tm_update = ? WHERE id = ?",
            delta, now, accountID.Bytes())
        if err != nil {
            return false, fmt.Errorf("could not update balance. err: %v", err)
        }

        // Update cost_total on the billing record we just inserted
        _, err = tx.ExecContext(ctx,
            "UPDATE billing_billings SET cost_total = ? WHERE id = ?",
            delta, b.ID.Bytes())
        if err != nil {
            return false, fmt.Errorf("could not update billing cost_total. err: %v", err)
        }
    }

    if err := tx.Commit(); err != nil {
        return false, err
    }

    // Invalidate caches after successful commit.
    // Without this, Redis would serve stale balance data until cache expires.
    _ = h.accountUpdateToCache(ctx, accountID)
    _ = h.billingUpdateToCache(ctx, b.ID)

    return true, nil
}
```

Return values:
- `(true, nil)` — record inserted and balance updated → success
- `(false, nil)` — duplicate key, record already existed → skip
- `(false, err)` — real error → log and continue to next account

### Billing Record Fields

| Field | Value |
|-------|-------|
| `id` | `h.utilHandler.UUIDCreate()` |
| `customer_id` | account's `CustomerID` (from embedded `commonidentity.Identity`) |
| `account_id` | account's `ID` |
| `reference_type` | `credit_free_tier` |
| `reference_id` | UUID v5 of `accountID:YYYY-MM` with `uuid.Nil` namespace |
| `cost_per_unit` | `0` |
| `cost_total` | initially `0`; updated to delta inside transaction if balance < $1.00 |
| `billing_unit_count` | `1.0` |
| `status` | `end` (completed immediately) |
| `tm_billing_start` | now |
| `tm_billing_end` | now |
| `tm_create` | now (auto-set) |
| `tm_update` | `nil` (auto-set) |
| `tm_delete` | `nil` |

## Files to Change

### New Files

**`bin-billing-manager/pkg/credithandler/`** — new package for free credit logic:
- `main.go` — handler struct, interface, constructor
- `credit.go` — `processAccount` method (builds billing struct, calls `BillingCreditTopUp`)
- `run.go` — `ProcessAll` method with pagination loop and daily ticker goroutine

Handler struct and constructor:
```go
type handler struct {
    db          dbhandler.DBHandler
    utilHandler utilhandler.UtilHandler
}

func NewCreditHandler(db dbhandler.DBHandler) CreditHandler {
    return &handler{
        db:          db,
        utilHandler: utilhandler.NewUtilHandler(),
    }
}
```

`processAccount` method:
```go
func (h *handler) processAccount(ctx context.Context, acc *account.Account) error {
    log := logrus.WithFields(logrus.Fields{"func": "processAccount", "account_id": acc.ID})

    // Generate deterministic reference_id for this account + month
    currentYearMonth := h.utilHandler.TimeNow().Format("2006-01")
    referenceID := h.utilHandler.NewV5UUID(uuid.Nil, acc.ID.String()+":"+currentYearMonth)

    now := h.utilHandler.TimeNow()
    b := &billing.Billing{
        Identity: commonidentity.Identity{
            ID:         h.utilHandler.UUIDCreate(),
            CustomerID: acc.CustomerID,
        },
        AccountID:        acc.ID,
        ReferenceType:    billing.ReferenceTypeCreditFreeTier,
        ReferenceID:      referenceID,
        CostPerUnit:      0,
        CostTotal:        0, // Updated inside transaction if credit is needed
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
```

### Modified Files

- `bin-billing-manager/models/billing/billing.go` — add `ReferenceTypeCreditFreeTier ReferenceType = "credit_free_tier"`
- `bin-billing-manager/pkg/dbhandler/billing.go`:
  - Add `BillingCreditTopUp(ctx, billing, accountID, targetAmount) (bool, error)` method (transactional insert + balance read with FOR UPDATE + balance update + cache invalidation)
  - Fix `BillingCreate` to set `tm_delete = nil` instead of sentinel `9999-01-01`
  - Remove `tmDeleteDefault` sentinel variable
  - Update `billingGetByReferenceTypeAndIDFromDB` query to use `tm_delete IS NULL` only (remove sentinel fallback from `sq.Or` clause)
  - Update any other queries that reference `tmDeleteDefault`
- `bin-billing-manager/pkg/dbhandler/billing_test.go`:
  - Update test expectations from `TMDelete: &tmDeleteDefault` to `TMDelete: nil`
  - Remove local `tmDeleteDefault` variable definitions in test functions
- `bin-billing-manager/pkg/dbhandler/main.go` — add `BillingCreditTopUp` to `DBHandler` interface:
  ```go
  BillingCreditTopUp(ctx context.Context, b *billing.Billing, accountID uuid.UUID, targetAmount float32) (bool, error)
  ```
- `bin-billing-manager/cmd/billing-manager/main.go` — initialize `credithandler`, launch daily ticker goroutine. Constants defined here:
  ```go
  const (
      creditCheckInterval = 24 * time.Hour
  )
  ```
- `bin-common-handler/pkg/utilhandler/` — add `NewV5UUID` to interface and implementation:
  ```go
  // Interface (main.go):
  NewV5UUID(namespace uuid.UUID, data string) uuid.UUID

  // Implementation (uuid.go or handler.go):
  func (h *handler) NewV5UUID(namespace uuid.UUID, data string) uuid.UUID {
      return uuid.NewV5(namespace, data)
  }
  ```

### bin-common-handler Change Impact

Adding `NewV5UUID` to `bin-common-handler/pkg/utilhandler` modifies a shared interface. This requires:
1. `go generate ./...` in `bin-common-handler` to regenerate mocks
2. `go mod tidy && go mod vendor` in ALL 30+ services to pick up the updated vendor

This is the standard bin-common-handler update workflow per CLAUDE.md.

### Database Migration Required

Add an Alembic migration in `bin-dbscheme-manager`:

1. **Clean up sentinel values FIRST** — normalize data before dedup to correctly identify active vs deleted records.
2. **Resolve any existing duplicates** — deduplicate records, preferring active records over deleted ones, then latest by `tm_create`.
3. **Add unique index** on `billing_billings(reference_type, reference_id)`.

```sql
-- Step 1: Clean up sentinel values first (normalize before dedup)
UPDATE billing_billings SET tm_delete = NULL
WHERE tm_delete = '9999-01-01 00:00:00.000000';

-- Step 2: Pre-check for duplicates
SELECT reference_type, reference_id, COUNT(*)
FROM billing_billings
GROUP BY reference_type, reference_id
HAVING COUNT(*) > 1;

-- Step 3: If duplicates exist, resolve (prefer active records, then latest):
DELETE b1 FROM billing_billings b1
INNER JOIN billing_billings b2
    ON b1.reference_type = b2.reference_type
    AND b1.reference_id = b2.reference_id
    AND (
        (b2.tm_delete IS NULL AND b1.tm_delete IS NOT NULL)
        OR (b1.tm_delete IS NULL AND b2.tm_delete IS NULL AND b1.tm_create < b2.tm_create)
        OR (b1.tm_delete IS NOT NULL AND b2.tm_delete IS NOT NULL AND b1.tm_create < b2.tm_create)
    );

-- Step 4: Add unique index for billing record deduplication
CREATE UNIQUE INDEX idx_billing_billings_reference_type_reference_id
ON billing_billings(reference_type, reference_id);
```

## Daily Ticker Design

- Interval: every 24 hours
- On startup: run immediately (catches up after deploys/restarts)
- Shuts down cleanly via `<-chDone` channel (matches existing pattern in `main.go`)

```go
// In cmd/billing-manager/main.go, after other goroutines:
go func() {
    // Run immediately on startup to catch up after deploys/restarts.
    if err := creditHandler.ProcessAll(context.Background()); err != nil {
        log.Errorf("Initial credit processing failed. err: %v", err)
    }

    ticker := time.NewTicker(creditCheckInterval)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            if err := creditHandler.ProcessAll(context.Background()); err != nil {
                log.Errorf("Credit processing failed. err: %v", err)
            }
        case <-chDone:
            return
        }
    }
}()
```

### Account Iteration

Use existing `AccountList` with cursor-based pagination. The method returns accounts ordered by `tm_create DESC` with a token for the next page (ISO8601 format). Loop until no more results:

```go
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
            break // all accounts processed
        }

        for _, acc := range accounts {
            if err := h.processAccount(ctx, acc); err != nil {
                log.Errorf("Failed to process credit for account. account_id: %s, err: %v", acc.ID, err)
                // continue to next account — don't block on individual failures
            }
        }

        // Use last account's tm_create as token for next page.
        // AccountList expects ISO8601Layout format (used by utilHandler.TimeGetCurTime).
        // TMCreate is *time.Time but always set by the database, so safe to dereference.
        token = accounts[len(accounts)-1].TMCreate.Format(utilhandler.ISO8601Layout)
    }

    return nil
}
```

## Constants

```go
// In credithandler package:
const (
    FreeTierCreditAmount float32 = 1.00 // Maximum free credit amount in USD
)

// In cmd/billing-manager/main.go (local to main, not exported):
const (
    creditCheckInterval = 24 * time.Hour
)
```

## Scope

- Applies to all existing and future free-tier accounts
- Only free-tier (`plan_type = "free"`) accounts are eligible
- Accounts on basic/professional/unlimited plans are excluded
