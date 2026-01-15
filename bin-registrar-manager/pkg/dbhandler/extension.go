package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-registrar-manager/models/extension"
)

const (
	extensionsTable = "registrar_extensions"
)

// extensionGetFromRow gets the extension from the row
func (h *handler) extensionGetFromRow(row *sql.Rows) (*extension.Extension, error) {
	res := &extension.Extension{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. extensionGetFromRow. err: %v", err)
	}

	return res, nil
}

// ExtensionCreate creates new Extension record.
func (h *handler) ExtensionCreate(ctx context.Context, b *extension.Extension) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	b.TMCreate = now
	b.TMUpdate = DefaultTimeStamp
	b.TMDelete = DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(b)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ExtensionCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(extensionsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ExtensionCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ExtensionCreate. err: %v", err)
	}

	// update the cache
	_ = h.extensionUpdateToCache(ctx, b.ID)

	return nil
}

// extensionGetFromDB returns Extension from the DB.
func (h *handler) extensionGetFromDB(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {
	fields := commondatabasehandler.GetDBFields(&extension.Extension{})
	query, args, err := squirrel.
		Select(fields...).
		From(extensionsTable).
		Where(squirrel.Eq{string(extension.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. extensionGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. extensionGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. extensionGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.extensionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. extensionGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// extensionUpdateToCache gets the extension from the DB and update the cache.
func (h *handler) extensionUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.extensionGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.extensionSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// extensionSetToCache sets the given extension to the cache
func (h *handler) extensionSetToCache(ctx context.Context, e *extension.Extension) error {
	if err := h.cache.ExtensionSet(ctx, e); err != nil {
		return err
	}

	return nil
}

// extensionGetFromCache returns Extension from the cache.
func (h *handler) extensionGetFromCache(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {

	// get from cache
	res, err := h.cache.ExtensionGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// extensionGetByEndpointIDFromCache returns Extension from the cache.
func (h *handler) extensionGetByEndpointIDFromCache(ctx context.Context, endpoint string) (*extension.Extension, error) {

	// get from cache
	res, err := h.cache.ExtensionGetByEndpointID(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// extensionGetByEndpointIDFromCache returns Extension from the cache.
func (h *handler) extensionGetByCustomerIDANDExtensionFromCache(ctx context.Context, customerID uuid.UUID, endpoint string) (*extension.Extension, error) {

	// get from cache
	res, err := h.cache.ExtensionGetByCustomerIDANDExtension(ctx, customerID, endpoint)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ExtensionGet returns extension.
func (h *handler) ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error) {

	res, err := h.extensionGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.extensionGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.extensionSetToCache(ctx, res)

	return res, nil
}

// ExtensionGetByEndpointID returns extension of the given endpoint.
func (h *handler) ExtensionGetByEndpointID(ctx context.Context, endpointID string) (*extension.Extension, error) {

	res, err := h.extensionGetByEndpointIDFromCache(ctx, endpointID)
	if err == nil {
		// Check if cached extension is deleted (soft delete check)
		if res.TMDelete >= commondatabasehandler.DefaultTimeStamp {
			return res, nil
		}
		// Cached extension is deleted, treat as not found
		res = nil
	}

	fields := commondatabasehandler.GetDBFields(&extension.Extension{})
	sb := squirrel.
		Select(fields...).
		From(extensionsTable).
		Where(squirrel.Eq{string(extension.FieldEndpointID): endpointID}).
		Where(squirrel.GtOrEq{string(extension.FieldTMDelete): commondatabasehandler.DefaultTimeStamp}).
		OrderBy(string(extension.FieldTMCreate) + " DESC").
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionGetByEndpointID. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionGetByEndpointID. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err = h.extensionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionGetByEndpointID. err: %v", err)
	}

	// set to the cache
	_ = h.extensionSetToCache(ctx, res)

	return res, nil
}

// ExtensionGetByExtension returns extension of the given extension.
func (h *handler) ExtensionGetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error) {

	res, err := h.extensionGetByCustomerIDANDExtensionFromCache(ctx, customerID, ext)
	if err == nil {
		// Check if cached extension is deleted (soft delete check)
		if res.TMDelete >= commondatabasehandler.DefaultTimeStamp {
			return res, nil
		}
		// Cached extension is deleted, treat as not found
		res = nil
	}

	fields := commondatabasehandler.GetDBFields(&extension.Extension{})
	sb := squirrel.
		Select(fields...).
		From(extensionsTable).
		Where(squirrel.Eq{string(extension.FieldCustomerID): customerID.Bytes()}).
		Where(squirrel.Eq{string(extension.FieldExtension): ext}).
		Where(squirrel.GtOrEq{string(extension.FieldTMDelete): commondatabasehandler.DefaultTimeStamp}).
		OrderBy(string(extension.FieldTMCreate) + " DESC").
		Limit(1).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. ExtensionGetByExtension. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionGetByExtension. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err = h.extensionGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan the row. ExtensionGetByExtension. err: %v", err)
	}

	// set to the cache
	_ = h.extensionSetToCache(ctx, res)

	return res, nil
}

// ExtensionDelete deletes given extension
func (h *handler) ExtensionDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[extension.Field]any{
		extension.FieldTMUpdate: ts,
		extension.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ExtensionDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(extensionsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(extension.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ExtensionDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ExtensionDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.extensionUpdateToCache(ctx, id)

	return nil
}

// ExtensionUpdate updates extension record with given fields.
func (h *handler) ExtensionUpdate(ctx context.Context, id uuid.UUID, fields map[extension.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[extension.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ExtensionUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(extensionsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(extension.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ExtensionUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ExtensionUpdate: exec failed: %w", err)
	}

	// update the cache
	_ = h.extensionUpdateToCache(ctx, id)

	return nil
}

// ExtensionGets returns list extensions.
func (h *handler) ExtensionGets(ctx context.Context, size uint64, token string, filters map[extension.Field]any) ([]*extension.Extension, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&extension.Extension{})
	sb := squirrel.
		Select(fields...).
		From(extensionsTable).
		Where(squirrel.Lt{string(extension.FieldTMCreate): token}).
		OrderBy(string(extension.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ExtensionGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ExtensionGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ExtensionGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*extension.Extension{}
	for rows.Next() {
		u, err := h.extensionGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("dbhandler: Could not scan the row. ExtensionGets. err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ExtensionGets. err: %v", err)
	}

	return res, nil
}
