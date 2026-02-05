package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

var (
	pipecatcallsTable = "pipecat_pipecatcalls"
)

// pipecatcallGetFromRow gets the pipecatcall from the row.
func (h *handler) pipecatcallGetFromRow(row *sql.Rows) (*pipecatcall.Pipecatcall, error) {
	res := &pipecatcall.Pipecatcall{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. pipecatcallGetFromRow. err: %v", err)
	}

	return res, nil
}

func (h *handler) PipecatcallCreate(ctx context.Context, f *pipecatcall.Pipecatcall) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	f.TMCreate = now
	f.TMUpdate = nil
	f.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(f)
	if err != nil {
		return fmt.Errorf("could not prepare fields. PipecatcallCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(pipecatcallsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. PipecatcallCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. PipecatcallCreate. err: %v", err)
	}

	_ = h.pipecatcallUpdateToCache(ctx, f.ID)
	return nil
}

func (h *handler) pipecatcallUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.pipecatcallGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.pipecatcallSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

func (h *handler) pipecatcallSetToCache(ctx context.Context, f *pipecatcall.Pipecatcall) error {
	if err := h.cache.PipecatcallSet(ctx, f); err != nil {
		return err
	}

	return nil
}

func (h *handler) pipecatcallGetFromCache(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {

	// get from cache
	res, err := h.cache.PipecatcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// pipecatcallGetFromDB gets the pipecatcall info from the db.
func (h *handler) pipecatcallGetFromDB(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {
	fields := commondatabasehandler.GetDBFields(&pipecatcall.Pipecatcall{})
	query, args, err := squirrel.
		Select(fields...).
		From(pipecatcallsTable).
		Where(squirrel.Eq{string(pipecatcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. pipecatcallGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. pipecatcallGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. pipecatcallGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.pipecatcallGetFromRow(row)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get data from row. pipecatcallGetFromDB. id: %s", id)
	}

	return res, nil
}

func (h *handler) PipecatcallGet(ctx context.Context, id uuid.UUID) (*pipecatcall.Pipecatcall, error) {

	res, err := h.pipecatcallGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.pipecatcallGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.pipecatcallSetToCache(ctx, res)

	return res, nil
}

func (h *handler) PipecatcallUpdate(ctx context.Context, id uuid.UUID, fields map[pipecatcall.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[pipecatcall.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("PipecatcallUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(pipecatcallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(pipecatcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("PipecatcallUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("PipecatcallUpdate: exec failed: %w", err)
	}

	_ = h.pipecatcallUpdateToCache(ctx, id)
	return nil
}

func (h *handler) PipecatcallDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[pipecatcall.Field]any{
		pipecatcall.FieldTMUpdate: ts,
		pipecatcall.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("PipecatcallDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(pipecatcallsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(pipecatcall.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("PipecatcallDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("PipecatcallDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.pipecatcallUpdateToCache(ctx, id)
	return nil
}
