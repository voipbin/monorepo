package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-storage-manager/models/file"
)

const (
	filesTable = "storage_files"
)

// fileGetFromRow gets the file from the row.
func (h *handler) fileGetFromRow(row *sql.Rows) (*file.File, error) {
	res := &file.File{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. fileGetFromRow. err: %v", err)
	}

	return res, nil
}

// FileCreate creates a new file row
func (h *handler) FileCreate(ctx context.Context, f *file.File) error {
	now := h.util.TimeGetCurTime()

	// Set timestamps
	f.TMCreate = now
	f.TMUpdate = commondatabasehandler.DefaultTimeStamp
	f.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(f)
	if err != nil {
		return fmt.Errorf("could not prepare fields. FileCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(filesTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. FileCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. FileCreate. err: %v", err)
	}

	_ = h.fileUpdateToCache(ctx, f.ID)

	return nil
}

// fileUpdateToCache gets the file from the DB and update the cache.
func (h *handler) fileUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.fileGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.fileSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// fileSetToCache sets the given file to the cache
func (h *handler) fileSetToCache(ctx context.Context, f *file.File) error {
	if err := h.cache.FileSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// fileGetFromCache returns file from the cache if possible.
func (h *handler) fileGetFromCache(ctx context.Context, id uuid.UUID) (*file.File, error) {
	// get from cache
	res, err := h.cache.FileGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// fileDeleteCache deletes cache
func (h *handler) fileDeleteCache(ctx context.Context, id uuid.UUID) error {
	// delete from cache
	err := h.cache.FileDel(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// fileGetFromDB gets the file info from the db.
func (h *handler) fileGetFromDB(ctx context.Context, id uuid.UUID) (*file.File, error) {
	fields := commondatabasehandler.GetDBFields(&file.File{})
	query, args, err := squirrel.
		Select(fields...).
		From(filesTable).
		Where(squirrel.Eq{string(file.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. fileGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. fileGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. fileGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.fileGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. fileGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// FileGet returns file.
func (h *handler) FileGet(ctx context.Context, id uuid.UUID) (*file.File, error) {
	res, err := h.fileGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.fileGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.fileSetToCache(ctx, res)

	return res, nil
}

// FileList returns files.
func (h *handler) FileList(ctx context.Context, token string, size uint64, filters map[file.Field]any) ([]*file.File, error) {
	if token == "" {
		token = h.util.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&file.File{})
	sb := squirrel.
		Select(fields...).
		From(filesTable).
		Where(squirrel.Lt{string(file.FieldTMCreate): token}).
		OrderBy(string(file.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. FileGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. FileGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. FileGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*file.File{}
	for rows.Next() {
		u, err := h.fileGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. FileGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. FileGets. err: %v", err)
	}

	return res, nil
}

// FileUpdate updates file fields.
func (h *handler) FileUpdate(ctx context.Context, id uuid.UUID, fields map[file.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[file.FieldTMUpdate] = h.util.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("FileUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(filesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(file.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("FileUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.Exec(sqlStr, args...); err != nil {
		return fmt.Errorf("FileUpdate: exec failed: %w", err)
	}

	// set to the cache
	_ = h.fileUpdateToCache(ctx, id)

	return nil
}

// FileDelete deletes the given file
func (h *handler) FileDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.util.TimeGetCurTime()

	fields := map[file.Field]any{
		file.FieldTMUpdate: ts,
		file.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("FileDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(filesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(file.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("FileDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("FileDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	// delete cache
	_ = h.fileDeleteCache(ctx, id)

	return nil
}
