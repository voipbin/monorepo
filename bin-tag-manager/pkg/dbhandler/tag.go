package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-tag-manager/models/tag"
)

const (
	// select query for tag get
	tagSelect = `
		select
			id,
			customer_id,

			name,
			detail,

			tm_create,
			tm_update,
			tm_delete
		from
			tags
		`
)

// tagGetFromRow gets the tag from the row.
func (h *handler) tagGetFromRow(row *sql.Rows) (*tag.Tag, error) {

	res := &tag.Tag{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Name,
		&res.Detail,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. tagGetFromRow. err: %v", err)
	}

	return res, nil
}

// TagCreate creates new tag record and returns the created tag record.
func (h *handler) TagCreate(ctx context.Context, a *tag.Tag) error {
	q := `insert into tags(
		id,
		customer_id,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?, ?, ?
	)
	`

	_, err := h.db.Exec(q,
		a.ID.Bytes(),
		a.CustomerID.Bytes(),

		a.Name,
		a.Detail,

		h.utilHandler.TimeGetCurTime(),
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute. TagCreate. err: %v", err)
	}

	// update the cache
	_ = h.tagUpdateToCache(ctx, a.ID)

	return nil
}

// tagUpdateToCache gets the tag from the DB and update the cache.
func (h *handler) tagUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.tagGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.tagSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// tagSetToCache sets the given tag to the cache
func (h *handler) tagSetToCache(ctx context.Context, u *tag.Tag) error {
	if err := h.cache.TagSet(ctx, u); err != nil {
		return err
	}

	return nil
}

// tagGetFromCache returns tag from the cache.
func (h *handler) tagGetFromCache(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {

	// get from cache
	res, err := h.cache.TagGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// tagGetFromDB returns tag from the DB.
func (h *handler) tagGetFromDB(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", tagSelect)

	row, err := h.db.Query(q, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. TagGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.tagGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("dbhandler: Could not scan the row. TagGetFromDB. err: %v", err)
	}

	return res, nil
}

// TagGet returns tag.
func (h *handler) TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error) {
	res, err := h.tagGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.tagGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.tagSetToCache(ctx, res)

	return res, nil
}

// TagGets returns tags.
func (h *handler) TagGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error) {
	// prepare
	q := fmt.Sprintf("%s where customer_id = ? and tm_create < ? order by tm_create desc limit ?", tagSelect)

	rows, err := h.db.Query(q, customerID.Bytes(), token, size)
	if err != nil {
		return nil, fmt.Errorf("could not query. TagGets. err: %v", err)
	}
	defer rows.Close()

	var res []*tag.Tag
	for rows.Next() {
		u, err := h.tagGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. TagGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// TagSetBasicInfo sets the tag's basic info.
func (h *handler) TagSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error {
	// prepare
	q := `
	update
		tags
	set
		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`
	_, err := h.db.Exec(q, name, detail, h.utilHandler.TimeGetCurTime(), id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. TagSetBasicInfo. err: %v", err)
	}

	// update the cache
	_ = h.tagUpdateToCache(ctx, id)

	return nil
}

// TagDelete delets the tag info.
func (h *handler) TagDelete(ctx context.Context, id uuid.UUID) error {
	// prepare
	q := `
	update
		tags
	set
		tm_update = ?,
		tm_delete = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	_, err := h.db.Exec(q, ts, ts, id.Bytes())
	if err != nil {
		return fmt.Errorf("could not execute. TagDelete. err: %v", err)
	}

	// update the cache
	_ = h.tagUpdateToCache(ctx, id)

	return nil
}
