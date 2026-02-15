package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-billing-manager/models/allowance"
)

const (
	allowancesTable = "billing_allowances"
)

// allowanceGetFromRow scans an allowance row.
func (h *handler) allowanceGetFromRow(row *sql.Rows) (*allowance.Allowance, error) {
	res := &allowance.Allowance{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. allowanceGetFromRow. err: %v", err)
	}

	return res, nil
}

// AllowanceCreate creates a new allowance record.
func (h *handler) AllowanceCreate(ctx context.Context, c *allowance.Allowance) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = c.TMCreate
	c.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("AllowanceCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(allowancesTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AllowanceCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateKey
		}
		return fmt.Errorf("AllowanceCreate: could not execute query. err: %v", err)
	}

	return nil
}

// AllowanceGet returns an allowance by ID.
func (h *handler) AllowanceGet(ctx context.Context, id uuid.UUID) (*allowance.Allowance, error) {
	cols := commondatabasehandler.GetDBFields(allowance.Allowance{})

	query, args, err := sq.Select(cols...).
		From(allowancesTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("AllowanceGet: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AllowanceGet: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.allowanceGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("AllowanceGet: could not scan row. err: %v", err)
	}

	return res, nil
}

// AllowanceGetCurrentByAccountID returns the active allowance cycle for the account.
func (h *handler) AllowanceGetCurrentByAccountID(ctx context.Context, accountID uuid.UUID) (*allowance.Allowance, error) {
	cols := commondatabasehandler.GetDBFields(allowance.Allowance{})

	query, args, err := sq.Select(cols...).
		From(allowancesTable).
		Where(sq.Eq{"account_id": accountID.Bytes()}).
		Where("cycle_start <= NOW()").
		Where("cycle_end > NOW()").
		Where(sq.Eq{"tm_delete": nil}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("AllowanceGetCurrentByAccountID: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AllowanceGetCurrentByAccountID: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.allowanceGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("AllowanceGetCurrentByAccountID: could not scan row. err: %v", err)
	}

	return res, nil
}

// AllowanceList returns a paginated list of allowances.
func (h *handler) AllowanceList(ctx context.Context, size uint64, token string, filters map[allowance.Field]any) ([]*allowance.Allowance, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(allowance.Allowance{})

	builder := sq.Select(cols...).
		From(allowancesTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AllowanceList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AllowanceList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AllowanceList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*allowance.Allowance{}
	for rows.Next() {
		u, err := h.allowanceGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("AllowanceList: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// AllowanceUpdate updates the allowance fields.
func (h *handler) AllowanceUpdate(ctx context.Context, id uuid.UUID, fields map[allowance.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("AllowanceUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(allowancesTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AllowanceUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AllowanceUpdate: could not execute. err: %v", err)
	}

	return nil
}

// AllowanceConsumeTokens atomically consumes tokens from the allowance and deducts credit overflow
// from the account balance within a single transaction.
// Returns the tokens actually consumed and the credit amount charged.
func (h *handler) AllowanceConsumeTokens(ctx context.Context, allowanceID uuid.UUID, accountID uuid.UUID, tokensNeeded int, creditPerUnit float32, tokenPerUnit int) (int, float32, error) {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("AllowanceConsumeTokens: could not begin transaction. err: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Lock allowance row and read current state.
	// Raw SQL: FOR UPDATE lock cannot be expressed via squirrel.
	var tokensTotal, tokensUsed int
	row := tx.QueryRowContext(ctx,
		fmt.Sprintf("SELECT tokens_total, tokens_used FROM %s WHERE id = ? FOR UPDATE", allowancesTable),
		allowanceID.Bytes())
	if err := row.Scan(&tokensTotal, &tokensUsed); err != nil {
		return 0, 0, fmt.Errorf("AllowanceConsumeTokens: could not read allowance. err: %v", err)
	}

	remaining := tokensTotal - tokensUsed
	if remaining < 0 {
		remaining = 0
	}

	var tokensConsumed int
	var creditCharged float32

	if remaining >= tokensNeeded {
		// Case A: Enough tokens
		tokensConsumed = tokensNeeded
		creditCharged = 0
	} else if remaining > 0 {
		// Case B: Partial tokens
		tokensConsumed = remaining
		overflowTokens := tokensNeeded - remaining
		if tokenPerUnit > 0 {
			creditCharged = (float32(overflowTokens) / float32(tokenPerUnit)) * creditPerUnit
		}
	} else {
		// Case C: No tokens
		tokensConsumed = 0
		if tokenPerUnit > 0 {
			creditCharged = (float32(tokensNeeded) / float32(tokenPerUnit)) * creditPerUnit
		}
	}

	// Update allowance tokens_used
	if tokensConsumed > 0 {
		now := h.utilHandler.TimeNow()
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE %s SET tokens_used = tokens_used + ?, tm_update = ? WHERE id = ?", allowancesTable),
			tokensConsumed, now, allowanceID.Bytes())
		if err != nil {
			return 0, 0, fmt.Errorf("AllowanceConsumeTokens: could not update allowance. err: %v", err)
		}
	}

	// Deduct credit overflow from account balance
	if creditCharged > 0 {
		// Lock account row
		var balance float32
		row := tx.QueryRowContext(ctx,
			fmt.Sprintf("SELECT balance FROM %s WHERE id = ? FOR UPDATE", accountsTable),
			accountID.Bytes())
		if err := row.Scan(&balance); err != nil {
			return 0, 0, fmt.Errorf("AllowanceConsumeTokens: could not read account balance. err: %v", err)
		}

		if balance < creditCharged {
			return 0, 0, ErrInsufficientBalance
		}

		now := h.utilHandler.TimeNow()
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE %s SET balance = balance - ?, tm_update = ? WHERE id = ?", accountsTable),
			creditCharged, now, accountID.Bytes())
		if err != nil {
			return 0, 0, fmt.Errorf("AllowanceConsumeTokens: could not subtract balance. err: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("AllowanceConsumeTokens: could not commit. err: %v", err)
	}

	// Invalidate caches after successful commit.
	if cacheErr := h.accountUpdateToCache(ctx, accountID); cacheErr != nil {
		logrus.WithField("account_id", accountID).Errorf("AllowanceConsumeTokens: could not update account cache. err: %v", cacheErr)
	}

	return tokensConsumed, creditCharged, nil
}
