package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-billing-manager/models/billing"
)

const (
	billingsTable = "billing_billings"
)

// billingGetFromRow gets the billing from the row.
func (h *handler) billingGetFromRow(row *sql.Rows) (*billing.Billing, error) {
	res := &billing.Billing{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. billingGetFromRow. err: %v", err)
	}

	return res, nil
}

// BillingCreate creates new billing record.
func (h *handler) BillingCreate(ctx context.Context, c *billing.Billing) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = nil
	c.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("BillingCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("BillingCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		// handle duplicate key error gracefully (MySQL error 1062)
		if isDuplicateKeyError(err) {
			return nil
		}
		return fmt.Errorf("BillingCreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, c.ID)

	return nil
}

// BillingCreditTopUp atomically inserts a billing record and tops up the account balance
// within a single transaction. The balance is read inside the transaction with FOR UPDATE
// to prevent concurrent modifications from producing incorrect deltas.
//
// Returns (true, nil) if the record was inserted and balance updated.
// Returns (false, nil) if the record already exists (duplicate key — already processed this month).
// Returns (false, err) on any other error.
func (h *handler) BillingCreditTopUp(ctx context.Context, b *billing.Billing, accountID uuid.UUID, targetAmount float32) (bool, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("BillingCreditTopUp: could not begin transaction. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Insert billing record
	b.TMCreate = h.utilHandler.TimeNow()
	b.TMUpdate = nil
	b.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(b)
	if err != nil {
		return false, fmt.Errorf("BillingCreditTopUp: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(billingsTable).SetMap(fields).ToSql()
	if err != nil {
		return false, fmt.Errorf("BillingCreditTopUp: could not build insert query. err: %v", err)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		if isDuplicateKeyError(err) {
			return false, nil // already processed this month
		}
		return false, fmt.Errorf("BillingCreditTopUp: could not insert billing. err: %v", err)
	}

	// 2. Lock account row and read current balance.
	// Raw SQL: FOR UPDATE lock cannot be expressed via squirrel.
	var balance float32
	row := tx.QueryRowContext(ctx, "SELECT balance FROM billing_accounts WHERE id = ? FOR UPDATE", accountID.Bytes())
	if err := row.Scan(&balance); err != nil {
		return false, fmt.Errorf("BillingCreditTopUp: could not read balance. err: %v", err)
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
			return false, fmt.Errorf("BillingCreditTopUp: could not update balance. err: %v", err)
		}

		// Update cost_total on the billing record we just inserted
		_, err = tx.ExecContext(ctx,
			"UPDATE billing_billings SET cost_total = ? WHERE id = ?",
			delta, b.ID.Bytes())
		if err != nil {
			return false, fmt.Errorf("BillingCreditTopUp: could not update billing cost_total. err: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("BillingCreditTopUp: could not commit. err: %v", err)
	}

	// Invalidate caches after successful commit.
	// Without this, Redis would serve stale balance data until cache expires.
	_ = h.accountUpdateToCache(ctx, accountID)
	_ = h.billingUpdateToCache(ctx, b.ID)

	return true, nil
}

// billingGetFromCache returns billing from the cache.
func (h *handler) billingGetFromCache(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {
	res, err := h.cache.BillingGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// billingGetByReferenceIDFromCache returns billing of the given reference id from the cache.
func (h *handler) billingGetByReferenceIDFromCache(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {
	res, err := h.cache.BillingGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// billingGetFromDB returns billing from the DB.
func (h *handler) billingGetFromDB(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {
	cols := commondatabasehandler.GetDBFields(billing.Billing{})

	query, args, err := sq.Select(cols...).
		From(billingsTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("billingGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("billingGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.billingGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("billingGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// billingGetByReferenceIDFromDB returns billing of the given reference id from the DB.
func (h *handler) billingGetByReferenceIDFromDB(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {
	cols := commondatabasehandler.GetDBFields(billing.Billing{})

	query, args, err := sq.Select(cols...).
		From(billingsTable).
		Where(sq.Eq{"reference_id": referenceID.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("billingGetByReferenceIDFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("billingGetByReferenceIDFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.billingGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("billingGetByReferenceIDFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// billingGetByReferenceTypeAndIDFromDB returns an active billing matching the reference type and id.
func (h *handler) billingGetByReferenceTypeAndIDFromDB(ctx context.Context, referenceType billing.ReferenceType, referenceID uuid.UUID) (*billing.Billing, error) {
	cols := commondatabasehandler.GetDBFields(billing.Billing{})

	query, args, err := sq.Select(cols...).
		From(billingsTable).
		Where(sq.Eq{
			"reference_type": string(referenceType),
			"reference_id":   referenceID.Bytes(),
		}).
		Where(sq.Expr("tm_delete IS NULL")).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("billingGetByReferenceTypeAndIDFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("billingGetByReferenceTypeAndIDFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.billingGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("billingGetByReferenceTypeAndIDFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// BillingGetByReferenceTypeAndID returns an active billing matching the reference type and id.
func (h *handler) BillingGetByReferenceTypeAndID(ctx context.Context, referenceType billing.ReferenceType, referenceID uuid.UUID) (*billing.Billing, error) {
	res, err := h.billingGetByReferenceTypeAndIDFromDB(ctx, referenceType, referenceID)
	if err != nil {
		return nil, err
	}

	_ = h.billingSetToCache(ctx, res)

	return res, nil
}

// billingUpdateToCache gets the billing from the DB and update the cache.
func (h *handler) billingUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.billingGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.billingSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// billingSetToCache sets the given billing to the cache
func (h *handler) billingSetToCache(ctx context.Context, c *billing.Billing) error {
	if err := h.cache.BillingSet(ctx, c); err != nil {
		return err
	}

	return nil
}

// BillingGet returns billing.
func (h *handler) BillingGet(ctx context.Context, id uuid.UUID) (*billing.Billing, error) {
	res, err := h.billingGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.billingGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.billingSetToCache(ctx, res)

	return res, nil
}

// BillingGetByReferenceID returns billing by reference id.
func (h *handler) BillingGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*billing.Billing, error) {
	res, err := h.billingGetByReferenceIDFromCache(ctx, referenceID)
	if err == nil {
		return res, nil
	}

	res, err = h.billingGetByReferenceIDFromDB(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.billingSetToCache(ctx, res)

	return res, nil
}

// BillingList returns a list of billing.
func (h *handler) BillingList(ctx context.Context, size uint64, token string, filters map[billing.Field]any) ([]*billing.Billing, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(billing.Billing{})

	builder := sq.Select(cols...).
		From(billingsTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("BillingList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("BillingList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("BillingList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*billing.Billing{}
	for rows.Next() {
		u, err := h.billingGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("BillingList: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// BillingUpdate updates the billing fields.
func (h *handler) BillingUpdate(ctx context.Context, id uuid.UUID, fields map[billing.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("BillingUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(billingsTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("BillingUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("BillingUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, id)

	return nil
}

// BillingSetStatusEnd sets the billing status to end
func (h *handler) BillingSetStatusEnd(ctx context.Context, id uuid.UUID, billingDuration float32, timestamp *time.Time) error {
	// prepare - using raw SQL for the formula
	q := `
	update
		billing_billings
	set
		status = ?,
		cost_total = cost_per_unit * ?,
		billing_unit_count = ?,
		tm_billing_end = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, billing.StatusEnd, billingDuration, billingDuration, timestamp, h.utilHandler.TimeNow(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. BillingSetStatusEnd. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, id)

	return nil
}

// BillingSetStatus sets the billing status
func (h *handler) BillingSetStatus(ctx context.Context, id uuid.UUID, status billing.Status) error {
	fields := map[billing.Field]any{
		billing.FieldStatus: status,
	}

	return h.BillingUpdate(ctx, id, fields)
}

// BillingDelete deletes the billing
func (h *handler) BillingDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(billingsTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("BillingDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("BillingDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.billingUpdateToCache(ctx, id)

	return nil
}

// isDuplicateKeyError checks if the error is a MySQL duplicate key error (1062)
// or a SQLite unique constraint violation.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "Duplicate entry") || strings.Contains(errMsg, "UNIQUE constraint failed")
}
