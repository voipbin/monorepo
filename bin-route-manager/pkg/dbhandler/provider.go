package dbhandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/provider"
)

const (
	// select query for provider get
	providerSelect = `
	select
		id,

		type,
		hostname,

		tech_prefix,
		tech_postfix,
		tech_headers,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	from
		route_providers
	`
)

// providerGetFromRow gets the provider from the row.
func (h *handler) providerGetFromRow(row *sql.Rows) (*provider.Provider, error) {
	var techHeaders string

	res := &provider.Provider{}
	if err := row.Scan(
		&res.ID,

		&res.Type,
		&res.Hostname,

		&res.TechPrefix,
		&res.TechPostfix,
		&techHeaders,

		&res.Name,
		&res.Detail,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. providerGetFromRow. err: %v", err)
	}

	if err := json.Unmarshal([]byte(techHeaders), &res.TechHeaders); err != nil {
		return nil, fmt.Errorf("could not unmarshal the data. providerGetFromRow. err: %v", err)
	}

	return res, nil
}

// ProviderCreate creates a new provider record
func (h *handler) ProviderCreate(ctx context.Context, p *provider.Provider) error {

	q := `insert into route_providers(
		id,

		type,
		hostname,

		tech_prefix,
		tech_postfix,
		tech_headers,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?,
		?, ?,
		?, ?, ?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. ProviderCreate. err: %v", err)
	}
	defer stmt.Close()

	techHeaders, err := json.Marshal(p.TechHeaders)
	if err != nil {
		return fmt.Errorf("could not marshal actions. ProviderCreate. err: %v", err)
	}

	ts := h.utilHandler.TimeGetCurTime()
	_, err = stmt.ExecContext(ctx,
		p.ID.Bytes(),

		p.Type,
		p.Hostname,

		p.TechPrefix,
		p.TechPostfix,
		techHeaders,

		p.Name,
		p.Detail,

		ts,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
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

	// prepare
	q := fmt.Sprintf("%s where id = ?", providerSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. providerGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. providerGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.providerGetFromRow(row)
	if err != nil {
		return nil, err
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
func (h *handler) ProviderGets(ctx context.Context, token string, limit uint64) ([]*provider.Provider, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, providerSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. ProviderGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var res []*provider.Provider
	for rows.Next() {
		u, err := h.providerGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. ProviderGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// ProviderDelete deletes the given provider
func (h *handler) ProviderDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update route_providers set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ProviderDelete. err: %v", err)
	}

	// delete cache
	_ = h.providerUpdateToCache(ctx, id)

	return nil
}

// ProviderUpdate updates the provider information.
func (h *handler) ProviderUpdate(ctx context.Context, p *provider.Provider) error {
	q := `
	update route_providers set
		type = ?,
		hostname = ?,

		tech_prefix = ?,
		tech_postfix = ?,
		tech_headers = ?,

		name = ?,
		detail = ?,
		tm_update = ?
	where
		id = ?
	`

	techHeaders, err := json.Marshal(p.TechHeaders)
	if err != nil {
		return err
	}

	ts := h.utilHandler.TimeGetCurTime()
	if _, err := h.db.Exec(q, p.Type, p.Hostname, p.TechPrefix, p.TechPostfix, techHeaders, p.Name, p.Detail, ts, p.ID.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. ProviderUpdate. err: %v", err)
	}

	// set to the cache
	_ = h.providerUpdateToCache(ctx, p.ID)

	return nil
}
