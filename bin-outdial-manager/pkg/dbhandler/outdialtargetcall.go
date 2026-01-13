package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
)

const (
	outdialtargetcallsTable = "outdial_outdialtargetcalls"
)

// outdialTargetCallGetFromRow gets the outdialtargetcall from the row.
func (h *handler) outdialTargetCallGetFromRow(row *sql.Rows) (*outdialtargetcall.OutdialTargetCall, error) {
	res := &outdialtargetcall.OutdialTargetCall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. outdialTargetCallGetFromRow. err: %v", err)
	}

	return res, nil
}

// OutdialTargetCallCreate insert a new outdialtargetcall record
func (h *handler) OutdialTargetCallCreate(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {
	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(t)
	if err != nil {
		return fmt.Errorf("could not prepare fields. OutdialTargetCallCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(outdialtargetcallsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. OutdialTargetCallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. OutdialTargetCallCreate. err: %v", err)
	}

	_ = h.outdialTargetCallUpdateToCache(ctx, t.ID)

	return nil
}

// outdialTargetCallUpdateToCache gets the outdialtargetcall from the DB and update the cache.
func (h *handler) outdialTargetCallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.outdialTargetCallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.outdialTargetCallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// outdialTargetCallSetToCache sets the given outdialTargetCall to the cache
func (h *handler) outdialTargetCallSetToCache(ctx context.Context, t *outdialtargetcall.OutdialTargetCall) error {
	if err := h.cache.OutdialTargetCallSet(ctx, t); err != nil {
		return err
	}

	if t.ActiveflowID != uuid.Nil {
		if errActiveflow := h.cache.OutdialTargetCallSetByActiveflowID(ctx, t); errActiveflow != nil {
			return errActiveflow
		}
	}

	if t.ReferenceID != uuid.Nil {
		if errReferenceID := h.cache.OutdialTargetCallSetByReferenceID(ctx, t); errReferenceID != nil {
			return errReferenceID
		}
	}

	return nil
}

// outdialTargetCallGetFromCache returns outdialTargetCall from the cache if possible.
func (h *handler) outdialTargetCallGetFromCache(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// get from cache
	res, err := h.cache.OutdialTargetCallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetCallGetFromCacheByActiveflowID returns outdialTargetCall from the cache if possible.
func (h *handler) outdialTargetCallGetFromCacheByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// get from cache
	res, err := h.cache.OutdialTargetCallGetByActiveflowID(ctx, activeflowID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetCallGetFromCacheByReferenceID returns outdialTargetCall from the cache if possible.
func (h *handler) outdialTargetCallGetFromCacheByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	// get from cache
	res, err := h.cache.OutdialTargetCallGetByReferenceID(ctx, referenceID)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// outdialTargetCallGetFromDB gets the outdialTargetCall info from the db.
func (h *handler) outdialTargetCallGetFromDB(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {
	fields := commondatabasehandler.GetDBFields(&outdialtargetcall.OutdialTargetCall{})
	query, args, err := squirrel.
		Select(fields...).
		From(outdialtargetcallsTable).
		Where(squirrel.Eq{string(outdialtargetcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. outdialTargetCallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. outdialTargetCallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. outdialTargetCallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetCallGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. outdialTargetCallGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// OutdialTargetCallGet returns outdialtargetcall.
func (h *handler) OutdialTargetCallGet(ctx context.Context, id uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	res, err := h.outdialTargetCallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.outdialTargetCallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetCallSetToCache(ctx, res)

	return res, nil
}

// OutdialTargetCallGetByReferenceID gets the outdialtargetcall by reference_id.
func (h *handler) OutdialTargetCallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	tmp, err := h.outdialTargetCallGetFromCacheByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	fields := commondatabasehandler.GetDBFields(&outdialtargetcall.OutdialTargetCall{})
	query, args, err := squirrel.
		Select(fields...).
		From(outdialtargetcallsTable).
		Where(squirrel.Eq{string(outdialtargetcall.FieldReferenceID): referenceID.Bytes()}).
		OrderBy(string(outdialtargetcall.FieldTMCreate) + " DESC").
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. OutdialTargetCallGetByReferenceID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGetByReferenceID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. OutdialTargetCallGetByReferenceID. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetCallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetCallSetToCache(ctx, res)

	return res, nil
}

// OutdialTargetCallGetByActiveflowID gets the outdialtargetcall by activeflow_id.
func (h *handler) OutdialTargetCallGetByActiveflowID(ctx context.Context, activeflowID uuid.UUID) (*outdialtargetcall.OutdialTargetCall, error) {

	tmp, err := h.outdialTargetCallGetFromCacheByActiveflowID(ctx, activeflowID)
	if err == nil {
		return tmp, nil
	}

	fields := commondatabasehandler.GetDBFields(&outdialtargetcall.OutdialTargetCall{})
	query, args, err := squirrel.
		Select(fields...).
		From(outdialtargetcallsTable).
		Where(squirrel.Eq{string(outdialtargetcall.FieldActiveflowID): activeflowID.Bytes()}).
		OrderBy(string(outdialtargetcall.FieldTMCreate) + " DESC").
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. OutdialTargetCallGetByActiveflowID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGetByActiveflowID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. OutdialTargetCallGetByActiveflowID. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.outdialTargetCallGetFromRow(row)
	if err != nil {
		return nil, err
	}

	_ = h.outdialTargetCallSetToCache(ctx, res)

	return res, nil
}

// OutdialTargetCallGets returns list of outdialtargetcalls.
func (h *handler) OutdialTargetCallGets(ctx context.Context, token string, size uint64, filters map[outdialtargetcall.Field]any) ([]*outdialtargetcall.OutdialTargetCall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&outdialtargetcall.OutdialTargetCall{})
	sb := squirrel.
		Select(fields...).
		From(outdialtargetcallsTable).
		Where(squirrel.Lt{string(outdialtargetcall.FieldTMCreate): token}).
		OrderBy(string(outdialtargetcall.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. OutdialTargetCallGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. OutdialTargetCallGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. OutdialTargetCallGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*outdialtargetcall.OutdialTargetCall{}
	for rows.Next() {
		u, err := h.outdialTargetCallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. OutdialTargetCallGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. OutdialTargetCallGets. err: %v", err)
	}

	return res, nil
}

// OutdialTargetCallUpdate updates the outdialtargetcall with given fields.
func (h *handler) OutdialTargetCallUpdate(ctx context.Context, id uuid.UUID, fields map[outdialtargetcall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[outdialtargetcall.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("OutdialTargetCallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(outdialtargetcallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(outdialtargetcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("OutdialTargetCallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("OutdialTargetCallUpdate: exec failed: %w", err)
	}

	_ = h.outdialTargetCallUpdateToCache(ctx, id)
	return nil
}
