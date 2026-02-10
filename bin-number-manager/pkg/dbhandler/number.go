package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-number-manager/models/number"
)

const (
	numbersTable = "number_numbers"
)

// numberGetFromRow gets the number from the row using commondatabasehandler.ScanRow.
func (h *handler) numberGetFromRow(row *sql.Rows) (*number.Number, error) {
	res := &number.Number{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. numberGetFromRow. err: %v", err)
	}

	return res, nil
}

// NumberCreate creates a new number record.
func (h *handler) NumberCreate(ctx context.Context, n *number.Number) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	n.TMPurchase = now
	n.TMRenew = now
	n.TMCreate = now
	n.TMUpdate = nil
	n.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(n)
	if err != nil {
		return fmt.Errorf("could not prepare fields. NumberCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(numbersTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. NumberCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. NumberCreate. err: %v", err)
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, n.ID)

	return nil
}

// NumberGetFromCacheByNumber returns number from the cache by number.
func (h *handler) NumberGetFromCacheByNumber(ctx context.Context, numb string) (*number.Number, error) {
	// get from cache
	res, err := h.cache.NumberGetByNumber(ctx, numb)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// numberGetFromCache returns number from the cache.
func (h *handler) numberGetFromCache(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	// get from cache
	res, err := h.cache.NumberGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// numberSetToCache sets the given number to the cache
func (h *handler) numberSetToCache(ctx context.Context, num *number.Number) error {
	if err := h.cache.NumberSet(ctx, num); err != nil {
		return err
	}

	return nil
}

// numberUpdateToCache gets the number from the DB and update the cache.
func (h *handler) numberUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.numberGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.numberSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// numberGetFromDB returns number info from the DB.
func (h *handler) numberGetFromDB(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	fields := commondatabasehandler.GetDBFields(&number.Number{})
	query, args, err := squirrel.
		Select(fields...).
		From(numbersTable).
		Where(squirrel.Eq{string(number.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. numberGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. numberGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. numberGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.numberGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get number. numberGetFromDB. id: %s", id)
	}

	return res, nil
}

// NumberGet returns number.
func (h *handler) NumberGet(ctx context.Context, id uuid.UUID) (*number.Number, error) {
	res, err := h.numberGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.numberGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.numberSetToCache(ctx, res)

	return res, nil
}

// NumberList returns a list of numbers.
func (h *handler) NumberList(ctx context.Context, size uint64, token string, filters map[number.Field]any) ([]*number.Number, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&number.Number{})
	sb := squirrel.
		Select(fields...).
		From(numbersTable).
		Where(squirrel.Lt{string(number.FieldTMCreate): token}).
		OrderBy(string(number.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. NumberList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. NumberList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberList, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. NumberList. err: %v", err)
	}

	return res, nil
}

// NumberUpdate updates a number with the given fields.
func (h *handler) NumberUpdate(ctx context.Context, id uuid.UUID, fields map[number.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[number.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("NumberUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(numbersTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(number.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("NumberUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("NumberUpdate: exec failed: %w", err)
	}

	_ = h.numberUpdateToCache(ctx, id)
	return nil
}

// NumberDelete sets the delete timestamp.
func (h *handler) NumberDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[number.Field]any{
		number.FieldStatus:   number.StatusDeleted,
		number.FieldTMUpdate: ts,
		number.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("NumberDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(numbersTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(number.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("NumberDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("NumberDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	// update the cache
	_ = h.numberUpdateToCache(ctx, id)

	return nil
}

// NumberGetExistingNumbers returns which of the given numbers already exist (active, not deleted) in the database.
func (h *handler) NumberGetExistingNumbers(ctx context.Context, numbers []string) ([]string, error) {
	if len(numbers) == 0 {
		return nil, nil
	}

	// build IN clause values
	vals := make([]any, len(numbers))
	for i, n := range numbers {
		vals[i] = n
	}

	query, args, err := squirrel.
		Select(string(number.FieldNumber)).
		From(numbersTable).
		Where(squirrel.Eq{string(number.FieldNumber): vals}).
		Where(squirrel.Eq{string(number.FieldStatus): string(number.StatusActive)}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. NumberGetExistingNumbers. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetExistingNumbers. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []string
	for rows.Next() {
		var num string
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("could not scan row. NumberGetExistingNumbers. err: %v", err)
		}
		res = append(res, num)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. NumberGetExistingNumbers. err: %v", err)
	}

	return res, nil
}

// NumberGetsByTMRenew returns a list of numbers by tm_renew.
func (h *handler) NumberGetsByTMRenew(ctx context.Context, tmRenew string, size uint64, filters map[number.Field]any) ([]*number.Number, error) {
	fields := commondatabasehandler.GetDBFields(&number.Number{})
	sb := squirrel.
		Select(fields...).
		From(numbersTable).
		Where(squirrel.Lt{string(number.FieldTMRenew): tmRenew}).
		OrderBy(string(number.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. NumberGetsByTMRenew. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. NumberGetsByTMRenew. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. NumberGetsByTMRenew. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*number.Number{}
	for rows.Next() {
		u, err := h.numberGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. NumberGetsByTMRenew, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. NumberGetsByTMRenew. err: %v", err)
	}

	return res, nil
}
