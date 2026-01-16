package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"
	"github.com/pkg/errors"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-conference-manager/models/conferencecall"
)

var (
	conferencecallTable = "conference_conferencecalls"
)

// conferencecallGetFromRow gets the conferencecall from the row.
func (h *handler) conferencecallGetFromRow(row *sql.Rows) (*conferencecall.Conferencecall, error) {
	res := &conferencecall.Conferencecall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. conferencecallGetFromRow. err: %v", err)
	}

	return res, nil
}

// ConferencecallCreate creates a new conferencecall record.
func (h *handler) ConferencecallCreate(ctx context.Context, cf *conferencecall.Conferencecall) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	cf.TMCreate = now
	cf.TMUpdate = commondatabasehandler.DefaultTimeStamp
	cf.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(cf)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ConferencecallCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(conferencecallTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ConferencecallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ConferencecallCreate. err: %v", err)
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
	fields := commondatabasehandler.GetDBFields(&conferencecall.Conferencecall{})
	query, args, err := squirrel.
		Select(fields...).
		From(conferencecallTable).
		Where(squirrel.Eq{string(conferencecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. conferencecallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. conferencecallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. conferencecallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.conferencecallGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. conferencecallGetFromDB. id: %s", id)
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

	fields := commondatabasehandler.GetDBFields(&conferencecall.Conferencecall{})
	query, args, err := squirrel.
		Select(fields...).
		From(conferencecallTable).
		Where(squirrel.Eq{string(conferencecall.FieldReferenceID): referenceID.Bytes()}).
		OrderBy(string(conferencecall.FieldTMCreate) + " DESC").
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ConferencecallGetByReferenceID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferencecallGetByReferenceID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

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
func (h *handler) ConferencecallList(ctx context.Context, size uint64, token string, filters map[conferencecall.Field]any) ([]*conferencecall.Conferencecall, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&conferencecall.Conferencecall{})
	sb := squirrel.
		Select(fields...).
		From(conferencecallTable).
		Where(squirrel.Lt{string(conferencecall.FieldTMCreate): token}).
		OrderBy(string(conferencecall.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ConferencecallGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ConferencecallGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ConferencecallGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*conferencecall.Conferencecall{}
	for rows.Next() {
		u, err := h.conferencecallGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ConferencecallGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ConferencecallGets. err: %v", err)
	}

	return res, nil
}

// ConferencecallUpdate updates the conferencecall with the given fields.
func (h *handler) ConferencecallUpdate(ctx context.Context, id uuid.UUID, fields map[conferencecall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[conferencecall.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ConferencecallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(conferencecallTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(conferencecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ConferencecallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ConferencecallUpdate: exec failed: %w", err)
	}

	// update the cache
	_ = h.conferencecallUpdateToCache(ctx, id)
	return nil
}

// ConferencecallDelete deletes the conferencecall
func (h *handler) ConferencecallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[conferencecall.Field]any{
		conferencecall.FieldTMUpdate: ts,
		conferencecall.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ConferencecallDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(conferencecallTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(conferencecall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ConferencecallDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ConferencecallDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.conferencecallUpdateToCache(ctx, id)

	return nil
}
