package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-outdial-manager/models/outdial"
)

const (
	outdialsTable = "outdial_outdials"
)

// outdialGetFromRow gets the outdial from the row.
func (h *handler) outdialGetFromRow(row *sql.Rows) (*outdial.Outdial, error) {
	res := &outdial.Outdial{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialGetFromRow. err: %v", err)
	}

	return res, nil
}

// OutdialCreate insert a new outdial record
func (h *handler) OutdialCreate(ctx context.Context, f *outdial.Outdial) error {
	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(f)
	if err != nil {
		return fmt.Errorf("could not prepare fields. OutdialCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(outdialsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. OutdialCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
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
	fields := commondatabasehandler.GetDBFields(&outdial.Outdial{})
	query, args, err := squirrel.
		Select(fields...).
		From(outdialsTable).
		Where(squirrel.Eq{string(outdial.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. outdialGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. outdialGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. outdialGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.outdialGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. outdialGetFromDB. id: %s, err: %v", id, err)
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
	ts := h.utilHandler.TimeNow()

	fields := map[outdial.Field]any{
		outdial.FieldTMUpdate: ts,
		outdial.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutdialDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(outdialsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outdial.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("OutdialDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("could not execute. OutdialDelete. err: %v", err)
	}

	// update the cache
	_ = h.outdialUpdateToCache(ctx, id)

	return nil
}

// OutdialList returns list of outdials.
func (h *handler) OutdialList(ctx context.Context, token string, size uint64, filters map[outdial.Field]any) ([]*outdial.Outdial, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&outdial.Outdial{})
	sb := squirrel.
		Select(fields...).
		From(outdialsTable).
		Where(squirrel.Lt{string(outdial.FieldTMCreate): token}).
		OrderBy(string(outdial.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. OutdialList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OutdialList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*outdial.Outdial{}
	for rows.Next() {
		u, err := h.outdialGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. OutdialList, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. OutdialList. err: %v", err)
	}

	return res, nil
}

// OutdialUpdate updates the outdial with given fields.
func (h *handler) OutdialUpdate(ctx context.Context, id uuid.UUID, fields map[outdial.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[outdial.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutdialUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(outdialsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outdial.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("OutdialUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("OutdialUpdate: exec failed: %w", err)
	}

	_ = h.outdialUpdateToCache(ctx, id)
	return nil
}
