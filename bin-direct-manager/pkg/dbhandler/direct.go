package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-direct-manager/models/direct"
)

const (
	directTable = "direct_directs"
)

// directGetFromRow scans a single row into a Direct struct using db tags
func (h *handler) directGetFromRow(rows *sql.Rows) (*direct.Direct, error) {
	res := &direct.Direct{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. directGetFromRow. err: %v", err)
	}

	return res, nil
}

// DirectCreate creates new direct record
func (h *handler) DirectCreate(ctx context.Context, d *direct.Direct) error {
	d.TMCreate = h.utilHandler.TimeNow()
	d.TMUpdate = nil

	// prepare fields for insert
	fields, err := commondatabasehandler.PrepareFields(d)
	if err != nil {
		return fmt.Errorf("could not prepare fields. DirectCreate. err: %v", err)
	}

	query, args, err := sq.Insert(directTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. DirectCreate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. DirectCreate. err: %v", err)
	}

	return nil
}

// DirectGet returns direct info from the DB.
func (h *handler) DirectGet(ctx context.Context, id uuid.UUID) (*direct.Direct, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&direct.Direct{})

	query, args, err := sq.Select(columns...).
		From(directTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. DirectGet. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. DirectGet. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.directGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. DirectGet. err: %v", err)
	}

	return res, nil
}

// DirectGetByHash returns direct info by hash from the DB.
func (h *handler) DirectGetByHash(ctx context.Context, hash string) (*direct.Direct, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&direct.Direct{})

	query, args, err := sq.Select(columns...).
		From(directTable).
		Where(sq.Eq{"hash": hash}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. DirectGetByHash. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. DirectGetByHash. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res, err := h.directGetFromRow(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. DirectGetByHash. err: %v", err)
	}

	return res, nil
}

// DirectGets returns directs based on filters.
func (h *handler) DirectGets(ctx context.Context, size uint64, token string, filters map[direct.Field]any) ([]*direct.Direct, error) {
	// get column names from db tags
	columns := commondatabasehandler.GetDBFields(&direct.Direct{})

	builder := sq.Select(columns...).
		From(directTable).
		Where("tm_create < ?", token).
		OrderBy("tm_create desc").
		Limit(size)

	// apply filters
	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. DirectGets. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. DirectGets. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. DirectGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*direct.Direct{}
	for rows.Next() {
		d, err := h.directGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. DirectGets. err: %v", err)
		}

		res = append(res, d)
	}

	return res, nil
}

// DirectUpdate updates a direct with the given fields.
func (h *handler) DirectUpdate(ctx context.Context, id uuid.UUID, fields map[direct.Field]any) error {
	// add update timestamp
	fields[direct.FieldTMUpdate] = h.utilHandler.TimeNow()

	// prepare fields for update
	data, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("could not prepare fields. DirectUpdate. err: %v", err)
	}

	query, args, err := sq.Update(directTable).
		SetMap(data).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. DirectUpdate. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. DirectUpdate. err: %v", err)
	}

	return nil
}

// DirectDelete hard-deletes the direct info.
func (h *handler) DirectDelete(ctx context.Context, id uuid.UUID) error {
	query, args, err := sq.Delete(directTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. DirectDelete. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("could not execute. DirectDelete. err: %v", err)
	}

	return nil
}
