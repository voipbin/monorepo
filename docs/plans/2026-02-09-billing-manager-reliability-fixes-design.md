# Billing Manager Reliability Fixes

Addresses three critical reliability gaps in bin-billing-manager: race conditions on balance operations, duplicate billing from event redelivery, and silent event processing failures.

## Fix 1: Atomic Balance Operations with Transaction + DB Constraint

### Problem

Balance operations in `pkg/dbhandler/account.go:224-269` have no locking or transaction boundaries. The check-then-charge pattern (`IsValidBalance` then `SubtractBalance`) is not atomic — concurrent charges can drain an account below zero. There is no database constraint preventing negative balances.

### Approach

Use `SELECT ... FOR UPDATE` within a database transaction for the check-and-deduct flow, plus a `CHECK(balance >= 0)` constraint as a safety net.

### Changes

**New method in `pkg/dbhandler/main.go` (interface):**

```go
AccountSubtractBalanceWithCheck(ctx context.Context, accountID uuid.UUID, amount float32) (*account.Account, error)
```

**Implementation in `pkg/dbhandler/account.go`:**

```go
func (h *handler) AccountSubtractBalanceWithCheck(ctx context.Context, accountID uuid.UUID, amount float32) (*account.Account, error) {
    tx, err := h.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, errors.Wrap(err, "could not begin transaction")
    }
    defer tx.Rollback()

    // Lock the row and read current balance
    row := tx.QueryRowContext(ctx,
        "SELECT balance FROM billing_accounts WHERE id = ? FOR UPDATE", accountID)
    var balance float32
    if err := row.Scan(&balance); err != nil {
        return nil, errors.Wrap(err, "could not read account balance")
    }

    // Check sufficient balance
    if balance < amount {
        return nil, ErrInsufficientBalance
    }

    // Deduct
    _, err = tx.ExecContext(ctx,
        "UPDATE billing_accounts SET balance = balance - ?, tm_update = ? WHERE id = ?",
        amount, time.Now().UTC(), accountID)
    if err != nil {
        return nil, errors.Wrap(err, "could not subtract balance")
    }

    if err := tx.Commit(); err != nil {
        return nil, errors.Wrap(err, "could not commit transaction")
    }

    _ = h.accountUpdateToCache(ctx, accountID)

    return h.AccountGet(ctx, accountID)
}
```

**Update `pkg/billinghandler/billing.go` (`BillingEnd`):**

Replace the separate `IsValidBalance` + `SubtractBalance` calls with a single `AccountSubtractBalanceWithCheck` call. Admin accounts bypass the check (existing behavior preserved).

**Update `pkg/accounthandler/balance.go` (`IsValidBalance`):**

No changes needed — `IsValidBalance` remains a read-only check used by other services via RPC. The atomicity is enforced at the charge point, not the check point.

**DB migration (`bin-dbscheme-manager`):**

```sql
-- upgrade
ALTER TABLE billing_accounts ADD CONSTRAINT chk_balance_non_negative CHECK (balance >= 0);

-- downgrade
ALTER TABLE billing_accounts DROP CONSTRAINT chk_balance_non_negative;
```

**New sentinel error in `pkg/dbhandler/`:**

```go
var ErrInsufficientBalance = errors.New("insufficient balance")
```

### Files to modify

- `bin-billing-manager/pkg/dbhandler/main.go` — Add `AccountSubtractBalanceWithCheck` to interface
- `bin-billing-manager/pkg/dbhandler/account.go` — Implement transactional method, add `ErrInsufficientBalance`
- `bin-billing-manager/pkg/billinghandler/billing.go` — Use new atomic method in `BillingEnd`
- `bin-dbscheme-manager/` — Add migration for CHECK constraint
- Regenerate mocks for dbhandler

---

## Fix 2: Idempotent Billing Creation with Application Check + DB Constraint

### Problem

Every event handler in `pkg/billinghandler/event.go` creates a new billing record with a fresh UUID without checking for duplicates. There is no unique constraint on `(reference_type, reference_id)`. RabbitMQ event redelivery causes double-charging.

### Approach

Add an application-level duplicate check before creating billing records, plus a database unique index as a safety net.

### Changes

**New method in `pkg/dbhandler/main.go` (interface):**

```go
BillingGetByReferenceTypeAndID(ctx context.Context, referenceType billing.ReferenceType, referenceID uuid.UUID) (*billing.Billing, error)
```

**Implementation in `pkg/dbhandler/billing.go`:**

Query `billing_billings` where `reference_type = ? AND reference_id = ? AND tm_delete = '9999-01-01 00:00:00.000000'`.

**Update `pkg/billinghandler/billing.go` (`BillingStart`):**

```go
func (h *billingHandler) BillingStart(...) (*billing.Billing, error) {
    // Idempotency check — return existing billing if already created
    existing, err := h.db.BillingGetByReferenceTypeAndID(ctx, referenceType, referenceID)
    if err == nil && existing != nil {
        log.WithField("billing", existing).Debugf(
            "Billing already exists for reference. Skipping creation. reference_type: %s, reference_id: %s",
            referenceType, referenceID)
        return existing, nil
    }

    // Proceed with creation as before
    ...
}
```

**Handle duplicate key error in `BillingCreate`:**

If the application check passes but a concurrent insert wins the race, the unique index will reject the second insert. Catch the MySQL duplicate key error (1062) and treat it as success — fetch and return the existing record.

```go
func (h *handler) BillingCreate(ctx context.Context, b *billing.Billing) error {
    _, err := h.db.ExecContext(ctx, ...)
    if err != nil {
        if isDuplicateKeyError(err) {
            return nil // Safe to ignore — record already exists
        }
        return errors.Wrap(err, "could not create billing")
    }
    return nil
}
```

**DB migration (`bin-dbscheme-manager`):**

```sql
-- upgrade
CREATE UNIQUE INDEX idx_billings_ref_type_id_active
    ON billing_billings (reference_type, reference_id, tm_delete);

-- downgrade
DROP INDEX idx_billings_ref_type_id_active ON billing_billings;
```

Note: The index includes `tm_delete` so that soft-deleted records don't conflict with new ones. A re-billing for the same reference after the old record is deleted will succeed.

### Files to modify

- `bin-billing-manager/pkg/dbhandler/main.go` — Add `BillingGetByReferenceTypeAndID` to interface
- `bin-billing-manager/pkg/dbhandler/billing.go` — Implement new query + handle duplicate key in `BillingCreate`
- `bin-billing-manager/pkg/billinghandler/billing.go` — Add idempotency check in `BillingStart`
- `bin-dbscheme-manager/` — Add migration for unique index
- Regenerate mocks for dbhandler

---

## Fix 3: Reliable Event Processing with Failed Event Persistence + Synchronous BillingEnd

### Problem

Two issues compound to create silent revenue loss:

1. `BillingEnd` runs in a fire-and-forget goroutine (`pkg/billinghandler/billing.go:70-78`). If it fails, the charge is silently dropped.
2. The subscribe handler (`pkg/subscribehandler/main.go:115-126`) always returns nil, even on error. Failed events are acked and lost forever. The underlying `ConsumeMessage` in `bin-common-handler` acks messages before processing, so nack/requeue is not possible without changes to the shared library.

### Approach

Make `BillingEnd` synchronous so errors propagate. For the subscribe handler, persist failed events to a database table and retry them with exponential backoff via a background loop.

### Part A: Synchronous BillingEnd

**Update `pkg/billinghandler/billing.go`:**

```go
// BEFORE
if flagEnd {
    go func() {
        if errBilling := h.BillingEnd(context.Background(), tmp, tmBillingStart, source, destination); errBilling != nil {
            log.Errorf("Could not end the billing. err: %v", errBilling)
        }
    }()
}

// AFTER
if flagEnd {
    if errBilling := h.BillingEnd(ctx, tmp, tmBillingStart, source, destination); errBilling != nil {
        return nil, errors.Wrap(errBilling, "could not end the billing")
    }
}
```

This means errors from SMS and number billing now propagate to the subscribe handler, which leads to Part B.

### Part B: Failed event persistence table

**DB migration (`bin-dbscheme-manager`):**

```sql
-- upgrade
CREATE TABLE billing_failed_events (
    id BINARY(16) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    event_publisher VARCHAR(255) NOT NULL,
    event_data JSON NOT NULL,
    error_message TEXT NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    next_retry_at DATETIME(6) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    tm_create DATETIME(6) NOT NULL,
    tm_update DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    INDEX idx_failed_events_status_retry (status, next_retry_at)
);

-- downgrade
DROP TABLE billing_failed_events;
```

### Part C: Failed event handler

**New package `pkg/failedeventhandler/`:**

```go
// main.go
type FailedEventHandler interface {
    Save(ctx context.Context, event *sock.Event, processingErr error) error
    RetryPending(ctx context.Context) error
}
```

- `Save` — Persists the failed event with exponential backoff schedule. Retry intervals: 1m, 5m, 25m, 2h, 10h.
- `RetryPending` — Queries events where `status IN ('pending', 'retrying') AND next_retry_at <= NOW()`. Reprocesses each event through the subscribe handler's `processEvent`. On success, deletes the record. On failure, increments `retry_count` and updates `next_retry_at`. Sets status to `exhausted` after max retries and logs at error level.

**Dependencies:**

The `FailedEventHandler` needs access to the subscribe handler's `processEvent` to replay events. To avoid circular dependencies, extract the event processing logic into a shared interface or pass it as a callback during construction.

```go
type EventProcessor func(event *sock.Event) error

func NewFailedEventHandler(db DBHandler, processor EventProcessor) FailedEventHandler
```

### Part D: Wire into subscribe handler

**Update `pkg/subscribehandler/main.go`:**

```go
func (h *subscribeHandler) processEventRun(m *sock.Event) error {
    if errProcess := h.processEvent(m); errProcess != nil {
        log.Errorf("Could not process event. Persisting for retry. err: %v", errProcess)
        if errSave := h.failedEventHandler.Save(context.Background(), m, errProcess); errSave != nil {
            log.Errorf("CRITICAL: Could not save failed event. Data loss possible. err: %v", errSave)
        }
    }
    return nil // Still ack — retry handled via the table
}
```

The subscribe handler constructor gains a `failedEventHandler` dependency.

### Part E: Retry loop

**Update `cmd/billing-manager/main.go`:**

Start a background goroutine that calls `RetryPending` every 60 seconds:

```go
go func() {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            if err := failedEventHandler.RetryPending(ctx); err != nil {
                log.Errorf("Failed event retry error: %v", err)
            }
        case <-ctx.Done():
            return
        }
    }
}()
```

### Prometheus metrics

- `billing_manager_failed_event_save_total` — Counter for persisted failures (labels: event_type, publisher)
- `billing_manager_failed_event_retry_total` — Counter for retry attempts (labels: result=success|failure)
- `billing_manager_failed_event_exhausted_total` — Counter for events that exhausted all retries (labels: event_type)

### Files to create

- `bin-billing-manager/pkg/failedeventhandler/main.go` — Interface + constructor
- `bin-billing-manager/pkg/failedeventhandler/handler.go` — Save and RetryPending implementation
- `bin-billing-manager/pkg/failedeventhandler/db.go` — DB operations for failed events table
- `bin-billing-manager/pkg/failedeventhandler/mock_main.go` — Generated mock

### Files to modify

- `bin-billing-manager/pkg/billinghandler/billing.go` — Remove goroutine, make BillingEnd synchronous
- `bin-billing-manager/pkg/subscribehandler/main.go` — Add failedEventHandler dependency, persist on error
- `bin-billing-manager/cmd/billing-manager/main.go` — Initialize failedEventHandler, start retry loop
- `bin-dbscheme-manager/` — Add migration for `billing_failed_events` table
- Regenerate mocks for subscribehandler (new dependency)

---

## Implementation Order

1. **Fix 1** first — Atomic balance operations. This is self-contained within dbhandler/billinghandler.
2. **Fix 2** second — Idempotency. Also self-contained, adds new dbhandler method and billinghandler check.
3. **Fix 3** last — Failed event handling. Depends on Fix 1 and Fix 2 being stable since retried events will go through the same billing creation and balance paths.

Each fix includes its own DB migration. All three migrations can be in the same Alembic revision or split into separate revisions.

## Testing Strategy

Each fix requires:
- Unit tests for new dbhandler methods (mocked DB)
- Unit tests for updated billinghandler logic (mocked dbhandler)
- Unit tests for failedeventhandler Save and RetryPending (mocked DB and processor)
- Update existing tests that depend on the goroutine behavior in BillingStart

Specific test cases:
- **Fix 1:** Test insufficient balance returns error, test concurrent subtract blocked by lock (integration), test admin account bypass
- **Fix 2:** Test duplicate event returns existing billing, test concurrent creation handled by DB constraint
- **Fix 3:** Test failed event is persisted, test retry succeeds and deletes record, test exhausted event logged, test BillingEnd error propagates from BillingStart
