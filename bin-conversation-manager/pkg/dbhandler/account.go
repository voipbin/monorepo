package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-conversation-manager/models/account"
)

var (
	accountsTable = "conversation_accounts"

	accountsFields = []string{
		string(account.FieldID),
		string(account.FieldCustomerID),
		string(account.FieldType),
		string(account.FieldName),
		string(account.FieldDetail),
		string(account.FieldSecret),
		string(account.FieldToken),
		string(account.FieldTMCreate),
		string(account.FieldTMUpdate),
		string(account.FieldTMDelete),
	}
)

// accountGetFromRow gets the account from the row.
func (h *handler) accountGetFromRow(row *sql.Rows) (*account.Account, error) {

	res := &account.Account{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Type,

		&res.Name,
		&res.Detail,

		&res.Secret,
		&res.Token,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. accountGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccountCreate creates a new account record
func (h *handler) AccountCreate(ctx context.Context, ac *account.Account) error {
	now := h.utilHandler.TimeGetCurTime()

	sb := squirrel.
		Insert(accountsTable).
		Columns(accountsFields...). // Spread if GetQuerySelectField returns []string
		Values(
			ac.ID.Bytes(),
			ac.CustomerID.Bytes(),
			ac.Type,
			ac.Name,
			ac.Detail,
			ac.Secret,
			ac.Token,
			now,                                    // tm_create
			commondatabasehandler.DefaultTimeStamp, // tm_update
			commondatabasehandler.DefaultTimeStamp, // tm_delete
		).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AccountCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. AccountCreate. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, ac.ID)
	return nil
}

// accountGetFromDB gets the account info from the db.
func (h *handler) accountGetFromDB(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	query, args, err := squirrel.
		Select(accountsFields...).
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
		// Wrap the error from accountGetFromRow to provide more context if desired,
		// or return it directly if it's already descriptive enough.
		// The example uses errors.Wrapf, let's follow that.
		return nil, errors.Wrapf(err, "could not get account from row. accountGetFromDB. id: %s", id)
	}

	return res, nil
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
func (h *handler) accountSetToCache(ctx context.Context, u *account.Account) error {
	if err := h.cache.AccountSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// accountGetFromCache returns account from the cache.
func (h *handler) accountGetFromCache(ctx context.Context, id uuid.UUID) (*account.Account, error) {

	// get from cache
	res, err := h.cache.AccountGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AccountGet returns the messagetarget.
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

// AccountGets returns a list of account.
func (h *handler) AccountGets(ctx context.Context, size uint64, token string, filters map[account.Field]any) ([]*account.Account, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	sb := squirrel.
		Select(accountsFields...).
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

// AccountSet returns sets the account info
func (h *handler) AccountSet(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) error {

	// prepare
	q := `
	update conversation_accounts set
		name = ?,
		detail = ?,
		secret = ?,
		token = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, name, detail, secret, token, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountSet. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountUpdate updates the account info.
func (h *handler) AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[account.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("AccountUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(accountsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{"id": id.Bytes()})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("AccountUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("AccountUpdate: exec failed: %w", err)
	}

	_ = h.accountUpdateToCache(ctx, id)
	return nil
}

func (h *handler) AccountDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

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
		return errors.Wrapf(err, "could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.accountUpdateToCache(ctx, id)
	return nil
}
