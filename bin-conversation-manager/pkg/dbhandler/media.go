package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
)

const (
	// select query for conversation get
	mediaSelect = `
	select
		id,
		customer_id,

		type,
		filename,

		tm_create,
		tm_update,
		tm_delete
	from
		conversation_medias
	`
)

// mediaGetFromRow gets the media from the row.
func (h *handler) mediaGetFromRow(row *sql.Rows) (*media.Media, error) {

	res := &media.Media{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.Type,
		&res.Filename,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. mediaGetFromRow. err: %v", err)
	}

	return res, nil
}

// MediaCreate creates a new media record
func (h *handler) MediaCreate(ctx context.Context, m *media.Media) error {

	q := `insert into conversation_medias(
		id,
		customer_id,

		type,
		filename,

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
		return fmt.Errorf("could not prepare. MediaCreate. err: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		m.ID.Bytes(),
		m.CustomerID.Bytes(),

		m.Type,
		m.Filename,

		m.TMCreate,
		m.TMUpdate,
		m.TMDelete,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. MediaCreate. err: %v", err)
	}

	_ = h.mediaUpdateToCache(ctx, m.ID)

	return nil
}

// mediaGetFromDB gets the media info from the db.
func (h *handler) mediaGetFromDB(ctx context.Context, id uuid.UUID) (*media.Media, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", mediaSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. mediaGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. mediaGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.mediaGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// mediaUpdateToCache gets the media from the DB and update the cache.
func (h *handler) mediaUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.mediaGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.mediaSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// mediaSetToCache sets the given media to the cache
func (h *handler) mediaSetToCache(ctx context.Context, flow *media.Media) error {
	if err := h.cache.MediaSet(ctx, flow); err != nil {
		return err
	}

	return nil
}

// mediaGetFromCache returns media from the cache.
func (h *handler) mediaGetFromCache(ctx context.Context, id uuid.UUID) (*media.Media, error) {

	// get from cache
	res, err := h.cache.MediaGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// MediaGet returns media.
func (h *handler) MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error) {

	res, err := h.mediaGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.mediaGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.mediaSetToCache(ctx, res)

	return res, nil
}
