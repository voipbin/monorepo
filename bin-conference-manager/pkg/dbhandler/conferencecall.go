package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	uuid "github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

const (
	conferencecallSelect = `
	select
		id,
		customer_id,
		conference_id,

		reference_type,
		reference_id,

		status,

		tm_create,
		tm_update,
		tm_delete

	from
		conferencecalls
	`
)

// conferencecallGetFromRow gets the conferencecall from the row.
func (h *handler) conferencecallGetFromRow(row *sql.Rows) (*conferencecall.Conferencecall, error) {

	res := &conferencecall.Conferencecall{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,
		&res.ConferenceID,

		&res.ReferenceType,
		&res.ReferenceID,

		&res.Status,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferencecallGetFromRow. err: %v", err)
	}

	return res, nil
}

// ConferencecallCreate creates a new conferencecall record.
func (h *handler) ConferencecallCreate(ctx context.Context, cf *conferencecall.Conferencecall) error {
	q := `insert into conferencecalls(
		id,
		customer_id,
		conference_id,

		reference_type,
		reference_id,

		status,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?, ?,
		?, ?,
		?,
		?, ?, ?
		)
	`

	_, err := h.db.Exec(q,
		cf.ID.Bytes(),
		cf.CustomerID.Bytes(),
		cf.ConferenceID.Bytes(),

		cf.ReferenceType,
		cf.ReferenceID.Bytes(),

		cf.Status,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. ConferencecallCreate. err: %v", err)
	}

	// update the cache
	_ = h.conferencecallUpdateToCache(ctx, cf.ID)

	return nil
}

// conferencecallGetFromCache returns conferencecall from the cache if possible.
func (h *handler) conferencecallGetFromCache(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {

	// get from cache
	res, err := h.cache.ConferencecallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// conferencecallGetFromDB gets conferencecall.
func (h *handler) conferencecallGetFromDB(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", conferencecallSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. conferencecallGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conferencecallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. conferencecallGetFromDB, err: %v", err)
	}

	return res, nil
}

// conferencecallUpdateToCache gets the conferencecall from the DB and update the cache.
func (h *handler) conferencecallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.conferencecallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.conferencecallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// conferencecallSetToCache sets the given conferencecall to the cache
func (h *handler) conferencecallSetToCache(ctx context.Context, data *conferencecall.Conferencecall) error {
	if err := h.cache.ConferencecallSet(ctx, data); err != nil {
		return err
	}

	return nil
}

// ConferencecallGet get conferencecall.
func (h *handler) ConferencecallGet(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {

	res, err := h.conferencecallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.conferencecallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.conferencecallSetToCache(ctx, res)

	return res, nil
}

// ConferencecallGetByReferenceID gets conferencecall of the given reference_id.
func (h *handler) ConferencecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error) {

	tmp, err := h.cache.ConferencecallGetByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	// prepare
	q := fmt.Sprintf("%s where reference_id = ? order by tm_create desc", conferencecallSelect)

	row, err := h.db.Query(q, referenceID.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferencecallGetByReferenceID. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.conferencecallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get call. ConferencecallGetByReferenceID, err: %v", err)
	}

	_ = h.conferencecallSetToCache(ctx, res)

	return res, nil
}

// ConferencecallGets returns a list of conferencecalls of the given filters.
func (h *handler) ConferencecallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conferencecall.Conferencecall, error) {

	// prepare
	q := fmt.Sprintf(`
			%s
		where
			tm_create < ?
	`, conferencecallSelect)

	values := []interface{}{
		token,
	}

	for k, v := range filters {
		switch k {
		case "customer_id", "conference_id", "reference_id":
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
		return nil, fmt.Errorf("could not query. ConferencecallGets. err: %v", err)
	}
	defer rows.Close()

	res := []*conferencecall.Conferencecall{}
	for rows.Next() {
		u, err := h.conferencecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ConferencecallGets, err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ConferencecallDelete deletes the conferencecall
func (h *handler) ConferencecallDelete(ctx context.Context, id uuid.UUID) error {
	//prepare
	q := `
	update conferencecalls set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferencecallDelete. err: %v", err)
	}

	// update the cache
	_ = h.conferencecallUpdateToCache(ctx, id)

	return nil
}

// ConferencecallUpdateStatus updates the conferencecall's status
func (h *handler) ConferencecallUpdateStatus(ctx context.Context, id uuid.UUID, status conferencecall.Status) error {
	//prepare
	q := `
	update conferencecalls set
		status = ?,
		tm_update = ?
	where
		id = ?
	`

	_, err := h.db.Exec(q, status, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. ConferencecallUpdateStatus. err: %v", err)
	}

	// update the cache
	_ = h.conferencecallUpdateToCache(ctx, id)

	return nil
}
