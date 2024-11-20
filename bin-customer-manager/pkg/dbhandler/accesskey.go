package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"monorepo/bin-customer-manager/models/accesskey"
	"strconv"

	"github.com/gofrs/uuid"
)

const (
	// select query for call get
	accesskeySelect = `
	select
		id,
		customer_id,

		name,
		detail,

		token,

		tm_expire,

		tm_create,
		tm_update,
		tm_delete
	from
		customer_accesskeys
	`
)

// accesskeyGetFromRow gets the accesskey from the row.
func (h *handler) accesskeyGetFromRow(row *sql.Rows) (*accesskey.Accesskey, error) {
	res := &accesskey.Accesskey{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.Token,

		&res.TMExpire,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. accesskeyGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccesskeyCreate creates new accesskey record and returns the created accesskey record.
func (h *handler) AccesskeyCreate(ctx context.Context, c *accesskey.Accesskey) error {
	q := `insert into customer_accesskeys(
		id,
		customer_id,

		name,
		detail,

		token,

		tm_expire,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, 
		?, 
		?, ?, ?
	)
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q,
		c.ID.Bytes(),
		c.CustomerID.Bytes(),

		c.Name,
		c.Detail,

		c.Token,

		c.TMExpire,

		ts,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. AccessKeyCreate. err: %v", err)
	}

	// update the cache
	_ = h.accesskeyUpdateToCache(ctx, c.ID)

	return nil
}

// accesskeyUpdateToCache gets the accesskey from the DB and update the cache.
func (h *handler) accesskeyUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.accesskeyGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.accesskeySetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// accesskeySetToCache sets the given accesskey to the cache
func (h *handler) accesskeySetToCache(ctx context.Context, u *accesskey.Accesskey) error {
	if err := h.cache.AccesskeySet(ctx, u); err != nil {
		return err
	}

	return nil
}

// accesskeyGetFromCache returns accesskey from the cache.
func (h *handler) accesskeyGetFromCache(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {

	// get from cache
	res, err := h.cache.AccesskeyGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accesskeyGetFromDB returns accesskey from the DB.
func (h *handler) accesskeyGetFromDB(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", accesskeySelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. accesskeyGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.accesskeyGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. accesskeyGetFromDB. err: %v", err)
	}

	return res, nil
}

// AccesskeyGet returns accesskey.
func (h *handler) AccesskeyGet(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {
	res, err := h.accesskeyGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.accesskeyGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.accesskeySetToCache(ctx, res)

	return res, nil
}

// AccesskeyGets returns accesskeys.
func (h *handler) AccesskeyGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*accesskey.Accesskey, error) {

	// prepare
	q := fmt.Sprintf(`%s
	where
		tm_create < ?
	`, accesskeySelect)

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

		default:
			q = fmt.Sprintf("%s and %s = ?", q, k)
			values = append(values, v)
		}
	}

	q = fmt.Sprintf("%s order by tm_create desc limit ?", q)
	values = append(values, strconv.FormatUint(size, 10))

	rows, err := h.db.Query(q, values...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AccesskeyGets. err: %v", err)
	}
	defer rows.Close()

	var res []*accesskey.Accesskey
	for rows.Next() {
		u, err := h.accesskeyGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. AccesskeyGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// AccesskeyDelete deletes the accesskey.
func (h *handler) AccesskeyDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		customer_accesskeys
	set
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccesskeyDelete. err: %v", err)
	}

	// update the cache
	_ = h.accesskeyUpdateToCache(ctx, id)

	return nil
}

// AccesskeySetBasicInfo sets the accesskey's basic info.
func (h *handler) AccesskeySetBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) error {
	// prepare
	q := `
	update
		customer_accesskeys
	set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, name, detail, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. AccessKeySetBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.accesskeyUpdateToCache(ctx, id)

	return nil
}
