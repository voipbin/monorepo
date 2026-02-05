package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-storage-manager/models/account"
)

const (
	accountsTable = "storage_accounts"
)

// accountGetFromRow gets the account from the row.
func (h *handler) accountGetFromRow(row *sql.Rows) (*account.Account, error) {
	res := &account.Account{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. accountGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccountCreate creates a new account row
func (h *handler) AccountCreate(ctx context.Context, a *account.Account) error {
	now := h.util.TimeNow()

	// Set timestamps
	a.TMCreate = now
	a.TMUpdate = nil
	a.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(a)
	if err != nil {
		return fmt.Errorf("could not prepare fields. AccountCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(accountsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AccountCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. AccountCreate. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, a.ID)

	return nil
}

// accountUpdateToCache gets the account from the DB and update the cache.
func (h *handler) accountUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.accountGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.accountSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// accountSetToCache sets the given account to the cache
func (h *handler) accountSetToCache(ctx context.Context, a *account.Account) error {
	if err := h.cache.AccountSet(ctx, a); err != nil {
		return err
	}

	return nil
}

// accountGetFromCache returns account from the cache if possible.
func (h *handler) accountGetFromCache(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	// get from cache
	res, err := h.cache.AccountGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accountGetFromDB gets the account info from the db.
func (h *handler) accountGetFromDB(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	fields := commondatabasehandler.GetDBFields(&account.Account{})
	query, args, err := squirrel.
		Select(fields...).
		From(accountsTable).
		Where(squirrel.Eq{string(account.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. accountGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. accountGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. accountGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.accountGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. accountGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// AccountGet returns account.
func (h *handler) AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	res, err := h.accountGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.accountGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.accountSetToCache(ctx, res)

	return res, nil
}

// AccountList returns list of accounts.
func (h *handler) AccountList(ctx context.Context, token string, size uint64, filters map[account.Field]any) ([]*account.Account, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&account.Account{})
	sb := squirrel.
		Select(fields...).
		From(accountsTable).
		Where(squirrel.Lt{string(account.FieldTMCreate): token}).
		OrderBy(string(account.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. AccountGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AccountGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AccountGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*account.Account{}
	for rows.Next() {
		u, err := h.accountGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AccountGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. AccountGets. err: %v", err)
	}

	return res, nil
}

// AccountUpdate updates account fields.
func (h *handler) AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[account.FieldTMUpdate] = h.util.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("AccountUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(accountsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(account.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("AccountUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("AccountUpdate: exec failed: %w", err)
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountIncreaseFileInfo increases the account's file info.
func (h *handler) AccountIncreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error {
	q := `
	update storage_accounts set
		total_file_count = total_file_count + ?,
		total_file_size = total_file_size + ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, filecount, filesize, h.util.TimeNow(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. AccountIncreaseFile. err: %v", err)
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountDecreaseFileInfo decreases the account's file info.
func (h *handler) AccountDecreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error {
	q := `
	update storage_accounts set
		total_file_count = total_file_count - ?,
		total_file_size = total_file_size - ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, filecount, filesize, h.util.TimeNow(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. AccountDecreaseFileInfo. err: %v", err)
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountDelete deletes the given account
func (h *handler) AccountDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.util.TimeNow()

	fields := map[account.Field]any{
		account.FieldTMUpdate: ts,
		account.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("AccountDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(accountsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(account.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("AccountDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("AccountDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}
