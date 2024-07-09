package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/account"
)

const (
	// select query for account get
	accountSelect = `
	select
		id,
		customer_id,

		type,

		name,
		detail,

		secret,
		token,

		tm_create,
		tm_update,
		tm_delete
	from
		conversation_accounts
	`
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

	q := `insert into conversation_accounts(
		id,
		customer_id,

		type,

		name,
		detail,

		secret,
		token,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?,
		?, ?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. AccountCreate. err: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		ac.ID.Bytes(),
		ac.CustomerID.Bytes(),

		ac.Type,

		ac.Name,
		ac.Detail,

		ac.Secret,
		ac.Token,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. AccountCreate. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, ac.ID)

	return nil
}

// accountGetFromDB gets the account info from the db.
func (h *handler) accountGetFromDB(ctx context.Context, id uuid.UUID) (*account.Account, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", accountSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. accountGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. accountGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.accountGetFromRow(row)
	if err != nil {
		return nil, err
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
func (h *handler) AccountGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*account.Account, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, accountSelect)

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

		case "customer_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AccountGets. err: %v", err)
	}
	defer rows.Close()

	res := []*account.Account{}
	for rows.Next() {
		u, err := h.accountGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AccountGets, err: %v", err)
		}

		res = append(res, u)
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

// AccountDelete deletes the call
func (h *handler) AccountDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update conversation_accounts set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccountDelete. err: %v", err)
	}

	// update the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}
