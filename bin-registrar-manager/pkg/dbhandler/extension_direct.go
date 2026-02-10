package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-registrar-manager/models/extensiondirect"
)

const (
	extensionDirectsTable = "registrar_directs"
)

// extensionDirectGetFromRow gets the extension direct from the row
func (h *handler) extensionDirectGetFromRow(row *sql.Rows) (*extensiondirect.ExtensionDirect, error) {
	res := &extensiondirect.ExtensionDirect{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. extensionDirectGetFromRow. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectCreate creates new ExtensionDirect record.
func (h *handler) ExtensionDirectCreate(ctx context.Context, ed *extensiondirect.ExtensionDirect) error {
	now := h.utilHandler.TimeNow()

	ed.TMCreate = now
	ed.TMUpdate = nil
	ed.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(ed)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ExtensionDirectCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(extensionDirectsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ExtensionDirectCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ExtensionDirectCreate. err: %v", err)
	}

	return nil
}

// extensionDirectGetFromDB returns ExtensionDirect from the DB.
func (h *handler) extensionDirectGetFromDB(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	query, args, err := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. extensionDirectGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. extensionDirectGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. extensionDirectGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.extensionDirectGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. extensionDirectGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// ExtensionDirectGet returns extension direct.
func (h *handler) ExtensionDirectGet(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	return h.extensionDirectGetFromDB(ctx, id)
}

// ExtensionDirectGetByExtensionID returns extension direct of the given extension ID.
func (h *handler) ExtensionDirectGetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error) {
	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	sb := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldExtensionID): extensionID.Bytes()}).
		Where(squirrel.Eq{string(extensiondirect.FieldTMDelete): nil}).
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionDirectGetByExtensionID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionDirectGetByExtensionID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.extensionDirectGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionDirectGetByExtensionID. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectGetByExtensionIDs returns extension directs for the given extension IDs.
func (h *handler) ExtensionDirectGetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error) {
	if len(extensionIDs) == 0 {
		return []*extensiondirect.ExtensionDirect{}, nil
	}

	ids := make([][]byte, len(extensionIDs))
	for i, id := range extensionIDs {
		ids[i] = id.Bytes()
	}

	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	sb := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldExtensionID): ids}).
		Where(squirrel.Eq{string(extensiondirect.FieldTMDelete): nil}).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionDirectGetByExtensionIDs. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionDirectGetByExtensionIDs. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*extensiondirect.ExtensionDirect{}
	for rows.Next() {
		ed, err := h.extensionDirectGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ExtensionDirectGetByExtensionIDs. err: %v", err)
		}
		res = append(res, ed)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ExtensionDirectGetByExtensionIDs. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectGetByHash returns extension direct of the given hash.
func (h *handler) ExtensionDirectGetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error) {
	fields := commondatabasehandler.GetDBFields(&extensiondirect.ExtensionDirect{})
	sb := squirrel.
		Select(fields...).
		From(extensionDirectsTable).
		Where(squirrel.Eq{string(extensiondirect.FieldHash): hash}).
		Where(squirrel.Eq{string(extensiondirect.FieldTMDelete): nil}).
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionDirectGetByHash. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionDirectGetByHash. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.extensionDirectGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionDirectGetByHash. err: %v", err)
	}

	return res, nil
}

// ExtensionDirectDelete deletes given extension direct (soft delete)
func (h *handler) ExtensionDirectDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[extensiondirect.Field]any{
		extensiondirect.FieldTMUpdate: ts,
		extensiondirect.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ExtensionDirectDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(extensionDirectsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(extensiondirect.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ExtensionDirectDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ExtensionDirectDelete: exec failed: %w", err)
	}

	return nil
}

// ExtensionDirectUpdate updates extension direct record with given fields.
func (h *handler) ExtensionDirectUpdate(ctx context.Context, id uuid.UUID, fields map[extensiondirect.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[extensiondirect.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ExtensionDirectUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(extensionDirectsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(extensiondirect.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ExtensionDirectUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ExtensionDirectUpdate: exec failed: %w", err)
	}

	return nil
}
