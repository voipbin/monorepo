package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-route-manager/models/provider"
)

const (
	providersTable = "route_providers"
)

// providerGetFromRow gets the provider from the row.
func (h *handler) providerGetFromRow(row *sql.Rows) (*provider.Provider, error) {
	res := &provider.Provider{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. providerGetFromRow. err: %v", err)
	}

	return res, nil
}

// ProviderCreate creates a new provider record
func (h *handler) ProviderCreate(ctx context.Context, p *provider.Provider) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	p.TMCreate = now
	p.TMUpdate = commondatabasehandler.DefaultTimeStamp
	p.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Initialize tech_headers if nil to avoid null in database
	if p.TechHeaders == nil {
		p.TechHeaders = map[string]string{}
	}

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(p)
	if err != nil {
		return fmt.Errorf("could not prepare fields. ProviderCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(providersTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. ProviderCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. ProviderCreate. err: %v", err)
	}

	_ = h.providerUpdateToCache(ctx, p.ID)

	return nil
}

// providerUpdateToCache gets the provider from the DB and update the cache.
func (h *handler) providerUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.providerGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.providerSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// providerSetToCache sets the given provider to the cache
func (h *handler) providerSetToCache(ctx context.Context, f *provider.Provider) error {
	if err := h.cache.ProviderSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// providerGetFromCache returns provider from the cache if possible.
func (h *handler) providerGetFromCache(ctx context.Context, id uuid.UUID) (*provider.Provider, error) {
	// get from cache
	res, err := h.cache.ProviderGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// providerGetFromDB gets the provider info from the db.
func (h *handler) providerGetFromDB(ctx context.Context, id uuid.UUID) (*provider.Provider, error) {
	fields := commondatabasehandler.GetDBFields(&provider.Provider{})
	query, args, err := squirrel.
		Select(fields...).
		From(providersTable).
		Where(squirrel.Eq{string(provider.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. providerGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. providerGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. providerGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.providerGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. providerGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// ProviderGet returns provider.
func (h *handler) ProviderGet(ctx context.Context, id uuid.UUID) (*provider.Provider, error) {
	res, err := h.providerGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.providerGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.providerSetToCache(ctx, res)

	return res, nil
}

// ProviderGets returns list of providers.
func (h *handler) ProviderGets(ctx context.Context, token string, limit uint64, filters map[provider.Field]any) ([]*provider.Provider, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&provider.Provider{})
	sb := squirrel.
		Select(fields...).
		From(providersTable).
		Where(squirrel.GtOrEq{string(provider.FieldTMDelete): commondatabasehandler.DefaultTimeStamp}).
		Where(squirrel.Lt{string(provider.FieldTMCreate): token}).
		OrderBy(string(provider.FieldTMCreate) + " DESC", string(provider.FieldID) + " DESC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. ProviderGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. ProviderGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. ProviderGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*provider.Provider{}
	for rows.Next() {
		u, err := h.providerGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. ProviderGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. ProviderGets. err: %v", err)
	}

	return res, nil
}

// ProviderDelete deletes the given provider
func (h *handler) ProviderDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[provider.Field]any{
		provider.FieldTMUpdate: ts,
		provider.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ProviderDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(providersTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(provider.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("ProviderDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("ProviderDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	// update cache after delete
	_ = h.providerUpdateToCache(ctx, id)

	return nil
}

// ProviderUpdate updates the provider information.
func (h *handler) ProviderUpdate(ctx context.Context, id uuid.UUID, fields map[provider.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[provider.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("ProviderUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(providersTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(provider.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("ProviderUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("ProviderUpdate: exec failed: %w", err)
	}

	_ = h.providerUpdateToCache(ctx, id)
	return nil
}
