package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/aicall"
)

const (
	aicallTable = "ai_aicalls"
)

// AIcallCreate creates a new aicall record.
func (h *handler) AIcallCreate(ctx context.Context, cb *aicall.AIcall) error {
	cb.TMEnd = nil
	cb.TMCreate = h.utilHandler.TimeNow()
	cb.TMUpdate = nil
	cb.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(cb)
	if err != nil {
		return fmt.Errorf("AIcallCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(aicallTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AIcallCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIcallCreate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, cb.ID)

	return nil
}

// aicallGetFromCache returns aicall from the cache if possible.
func (h *handler) aicallGetFromCache(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.cache.AIcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// aicallGetFromDB gets aicall from the database.
func (h *handler) aicallGetFromDB(id uuid.UUID) (*aicall.AIcall, error) {
	cols := commondatabasehandler.GetDBFields(aicall.AIcall{})

	query, args, err := sq.Select(cols...).
		From(aicallTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("aicallGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("aicallGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &aicall.AIcall{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("aicallGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// aicallUpdateToCache gets the aicall from the DB and updates the cache.
func (h *handler) aicallUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.aicallGetFromDB(id)
	if err != nil {
		return err
	}

	if err := h.aicallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// aicallSetToCache sets the given aicall to the cache.
func (h *handler) aicallSetToCache(ctx context.Context, data *aicall.AIcall) error {
	if err := h.cache.AIcallSet(ctx, data); err != nil {
		return err
	}

	return nil
}

// AIcallGet gets aicall.
func (h *handler) AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
	res, err := h.aicallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.aicallGetFromDB(id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}

// AIcallGetByReferenceID gets aicall of the given reference_id.
func (h *handler) AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error) {
	tmp, err := h.cache.AIcallGetByReferenceID(ctx, referenceID)
	if err == nil {
		return tmp, nil
	}

	cols := commondatabasehandler.GetDBFields(aicall.AIcall{})

	query, args, err := sq.Select(cols...).
		From(aicallTable).
		Where(sq.Eq{"reference_id": referenceID.Bytes()}).
		OrderBy("tm_create desc").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIcallGetByReferenceID: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIcallGetByReferenceID: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &aicall.AIcall{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("AIcallGetByReferenceID: could not scan row. err: %v", err)
	}

	_ = h.aicallSetToCache(ctx, res)

	return res, nil
}

// AIcallDelete deletes the aicall.
func (h *handler) AIcallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(aicallTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIcallDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIcallDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}

// AIcallGets returns a list of aicalls.
func (h *handler) AIcallList(ctx context.Context, size uint64, token string, filters map[aicall.Field]any) ([]*aicall.AIcall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(aicall.AIcall{})

	builder := sq.Select(cols...).
		From(aicallTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AIcallList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIcallList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIcallList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*aicall.AIcall{}
	for rows.Next() {
		u := &aicall.AIcall{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("AIcallList: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// AIcallUpdate updates the aicall fields.
func (h *handler) AIcallUpdate(ctx context.Context, id uuid.UUID, fields map[aicall.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("AIcallUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(aicallTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIcallUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIcallUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aicallUpdateToCache(ctx, id)

	return nil
}
