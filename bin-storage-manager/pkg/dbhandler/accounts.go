package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"monorepo/bin-storage-manager/models/account"
	"strconv"

	"github.com/gofrs/uuid"
)

const (
	// select query for account get
	accountSelect = `
	select
		id,
		customer_id,

		total_file_count,
		total_file_size,

		tm_create,
		tm_update,
		tm_delete
	from
		storage_accounts
	`
)

// accountGetFromRow gets the file from the row.
func (h *handler) accountGetFromRow(row *sql.Rows) (*account.Account, error) {

	res := &account.Account{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.TotalFileCount,
		&res.TotalFileSize,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. accountGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccountCreate creates a new account row
func (h *handler) AccountCreate(ctx context.Context, f *account.Account) error {

	q := `insert into storage_accounts(
		id,
		customer_id,

		total_file_count,
		total_file_size,

        tm_create,
        tm_update,
        tm_delete
	) values(
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
		f.ID.Bytes(),
		f.CustomerID.Bytes(),

		f.TotalFileCount,
		f.TotalFileSize,

		h.util.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. AccountCreate. err: %v", err)
	}

	_ = h.accountUpdateToCache(ctx, f.ID)

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

// accountSetToCache sets the given file to the cache
func (h *handler) accountSetToCache(ctx context.Context, f *account.Account) error {
	if err := h.cache.AccountSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// accountGetFromCache returns file from the cache if possible.
func (h *handler) accountGetFromCache(ctx context.Context, id uuid.UUID) (*account.Account, error) {

	// get from cache
	res, err := h.cache.AccountGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accountGetFromDB gets the file info from the db.
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

// AccountGets returns list of accounts.
func (h *handler) AccountGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*account.Account, error) {
	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, accountSelect)

	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id":
			q = fmt.Sprintf("%s and %s = ?", q, k)
			tmp := uuid.FromStringOrNil(v)
			values = append(values, tmp.Bytes())

		case "deleted":
			if v == "false" {
				q = fmt.Sprintf("%s and tm_delete >= ?", q)
				values = append(values, DefaultTimeStamp)
			}

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
			return nil, fmt.Errorf("dbhandler: Could not scan the row. AccountGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// AccountIncreaseFileInfo increase the account info.
func (h *handler) AccountIncreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error {
	q := `
	update storage_accounts set
		total_file_count = total_file_count + ?,
		total_file_size = total_file_size + ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, filecount, filesize, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. AccountIncreaseFile. err: %v", err)
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountDecreaseFileInfo increase the account info.
func (h *handler) AccountDecreaseFileInfo(ctx context.Context, id uuid.UUID, filecount int64, filesize int64) error {
	q := `
	update storage_accounts set
		total_file_count = total_file_count - ?,
		total_file_size = total_file_size - ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, filecount, filesize, h.util.TimeGetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. AccountDecreaseFileInfo. err: %v", err)
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}

// AccountDelete deletes the given account
func (h *handler) AccountDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update storage_accounts set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.util.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. AccountDelete. err: %v", err)
	}

	// set to the cache
	_ = h.accountUpdateToCache(ctx, id)

	return nil
}
