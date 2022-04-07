package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
)

const (
	// select query for outdial get
	outdialSelect = `
	select
		id,
		customer_id,
		campaign_id,

		name,
		detail,

		data,

		tm_create,
		tm_update,
		tm_delete
	from
		outdials
	`
)

// outdialGetFromRow gets the outdial from the row.
func (h *handler) outdialGetFromRow(row *sql.Rows) (*outdial.Outdial, error) {
	res := &outdial.Outdial{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.CampaignID,

		&res.Name,
		&res.Detail,

		&res.Data,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialGetFromRow. err: %v", err)
	}

	return res, nil
}

// OutdialCreate insert a new outdial record
func (h *handler) OutdialCreate(ctx context.Context, f *outdial.Outdial) error {

	q := `insert into outdials(
		id,
		customer_id,
		campaign_id,

		name,
		detail,

		data,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. OutdialCreate. err: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		f.ID.Bytes(),
		f.CustomerID.Bytes(),
		f.CampaignID.Bytes(),

		f.Name,
		f.Detail,

		f.Data,

		f.TMCreate,
		f.TMUpdate,
		f.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. OutdialCreate. err: %v", err)
	}

	_ = h.outdialUpdateToCache(ctx, f.ID)

	return nil
}

// outdialUpdateToCache gets the outdial from the DB and update the cache.
func (h *handler) outdialUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.outdialGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.outdialSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// outdialSetToCache sets the given outdial to the cache
func (h *handler) outdialSetToCache(ctx context.Context, f *outdial.Outdial) error {
	if err := h.cache.OutdialSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// outdialGetFromCache returns outdial from the cache if possible.
func (h *handler) outdialGetFromCache(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error) {

	// get from cache
	res, err := h.cache.OutdialGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialGetFromDB gets the outdial info from the db.
func (h *handler) outdialGetFromDB(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", outdialSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. outdialGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. outdialGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.outdialGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OutdialGet returns outdial.
func (h *handler) OutdialGet(ctx context.Context, id uuid.UUID) (*outdial.Outdial, error) {

	res, err := h.outdialGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.outdialGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.outdialSetToCache(ctx, res)

	return res, nil
}

// OutdialDelete deletes the outdial.
func (h *handler) OutdialDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		outdials
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := GetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. OutdialDelete. err: %v", err)
	}

	// update the cache
	_ = h.outdialUpdateToCache(ctx, id)

	return nil
}

// OutdialGetsByCustomerID returns list of outdials.
func (h *handler) OutdialGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*outdial.Outdial, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, outdialSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*outdial.Outdial
	for rows.Next() {
		u, err := h.outdialGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. OutdialGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// OutdialUpdateBasicInfo updates outdial's basic information.
func (h *handler) OutdialUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	q := `
	update outdials set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, name, detail, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialUpdateBasicInfo. err: %v", err)
	}

	// set to the cache
	_ = h.outdialUpdateToCache(ctx, id)

	return nil
}

// OutdialUpdateCampaignID updates outdial's campaign.
func (h *handler) OutdialUpdateCampaignID(ctx context.Context, id, campaignID uuid.UUID) error {
	q := `
	update outdials set
		campaign_id = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, campaignID.Bytes(), GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialUpdateCampaignID. err: %v", err)
	}

	// set to the cache
	_ = h.outdialUpdateToCache(ctx, id)

	return nil
}

// OutdialUpdateData updates outdial's data.
func (h *handler) OutdialUpdateData(ctx context.Context, id uuid.UUID, data string) error {
	q := `
	update outdials set
		data = ?,
		tm_update = ?
	where
		id = ?
	`

	if _, err := h.db.Exec(q, data, GetCurTime(), id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. OutdialUpdateData. err: %v", err)
	}

	// set to the cache
	_ = h.outdialUpdateToCache(ctx, id)

	return nil
}
