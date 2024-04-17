package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/route"
)

const (
	// select query for route get
	routeSelect = `
	select
		id,
		customer_id,

		provider_id,
		priority,

		target,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	from
		routes
	`
)

// routeGetFromRow gets the route from the row.
func (h *handler) routeGetFromRow(row *sql.Rows) (*route.Route, error) {
	res := &route.Route{}
	if err := row.Scan(
		&res.ID,
		&res.CustomerID,

		&res.ProviderID,
		&res.Priority,

		&res.Target,

		&res.Name,
		&res.Detail,

		&res.TMCreate,
		&res.TMUpdate,
		&res.TMDelete,
	); err != nil {
		return nil, fmt.Errorf("could not scan the row. providerGetFromRow. err: %v", err)
	}

	return res, nil
}

// RouteCreate creates a new route record
func (h *handler) RouteCreate(ctx context.Context, r *route.Route) error {

	q := `insert into routes(
		id,
		customer_id,

		provider_id,
		priority,

		target,

		name,
		detail,

		tm_create,
		tm_update,
		tm_delete
	) values(
		?, ?,
		?, ?,
		?,
		?, ?,
		?, ?, ?
		)`
	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("could not prepare. RouteCreate. err: %v", err)
	}
	defer stmt.Close()

	ts := h.utilHandler.TimeGetCurTime()
	_, err = stmt.ExecContext(ctx,
		r.ID.Bytes(),
		r.CustomerID.Bytes(),

		r.ProviderID.Bytes(),
		r.Priority,

		r.Target,

		r.Name,
		r.Detail,

		ts,
		DefaultTimeStamp,
		DefaultTimeStamp,
	)
	if err != nil {
		return fmt.Errorf("could not execute query. ProviderCreate. err: %v", err)
	}

	_ = h.routeUpdateToCache(ctx, r.ID)

	return nil
}

// providerUpdateToCache gets the provider from the DB and update the cache.
func (h *handler) routeUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.routeGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.routeSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// providerSetToCache sets the given provider to the cache
func (h *handler) routeSetToCache(ctx context.Context, f *route.Route) error {
	if err := h.cache.RouteSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// providerGetFromCache returns provider from the cache if possible.
func (h *handler) routeGetFromCache(ctx context.Context, id uuid.UUID) (*route.Route, error) {

	// get from cache
	res, err := h.cache.RouteGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// routeGetFromDB gets the route info from the db.
func (h *handler) routeGetFromDB(ctx context.Context, id uuid.UUID) (*route.Route, error) {

	// prepare
	q := fmt.Sprintf("%s where id = ?", routeSelect)

	stmt, err := h.db.PrepareContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("could not prepare. routeGetFromDB. err: %v", err)
	}
	defer stmt.Close()

	// query
	row, err := stmt.QueryContext(ctx, id.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not query. routeGetFromDB. err: %v", err)
	}
	defer row.Close()

	if !row.Next() {
		return nil, ErrNotFound
	}

	res, err := h.routeGetFromRow(row)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// RouteGet returns route.
func (h *handler) RouteGet(ctx context.Context, id uuid.UUID) (*route.Route, error) {

	res, err := h.routeGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.routeGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = h.routeSetToCache(ctx, res)

	return res, nil
}

// RouteGets returns list of routes.
func (h *handler) RouteGets(ctx context.Context, token string, limit uint64) ([]*route.Route, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, routeSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. RouteGets. err: %v", err)
	}
	defer rows.Close()

	var res []*route.Route
	for rows.Next() {
		u, err := h.routeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. RouteGets. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// RouteGetsByCustomerID returns list of routes.
func (h *handler) RouteGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*route.Route, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and tm_create < ?
		order by
			tm_create desc, id desc
		limit ?
	`, routeSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), token, limit)
	if err != nil {
		return nil, fmt.Errorf("could not query. RouteGetsByCustomerID. err: %v", err)
	}
	defer rows.Close()

	var res []*route.Route
	for rows.Next() {
		u, err := h.routeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. RouteGetsByCustomerID. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// RouteGetsByCustomerIDWithTarget returns list of routes.
func (h *handler) RouteGetsByCustomerIDWithTarget(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error) {

	// prepare
	q := fmt.Sprintf(`
		%s
		where
			tm_delete >= ?
			and customer_id = ?
			and target = ?
		order by
			priority asc
	`, routeSelect)

	rows, err := h.db.Query(q, DefaultTimeStamp, customerID.Bytes(), target)
	if err != nil {
		return nil, fmt.Errorf("could not query. RouteGetsByCustomerIDWithTarget. err: %v", err)
	}
	defer rows.Close()

	var res []*route.Route
	for rows.Next() {
		u, err := h.routeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not scan the row. RouteGetsByCustomerIDWithTarget. err: %v", err)
		}

		res = append(res, u)
	}

	return res, nil
}

// RouteDelete deletes the given route
func (h *handler) RouteDelete(ctx context.Context, id uuid.UUID) error {
	q := `
	update routes set
		tm_delete = ?,
		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	if _, err := h.db.Exec(q, ts, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. RouteDelete. err: %v", err)
	}

	_ = h.cache.RouteDelete(ctx, id)

	return nil
}

// RouteUpdate updates the route information.
func (h *handler) RouteUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	providerID uuid.UUID,
	priority int,
	target string,
) error {
	q := `
	update routes set
		name = ?,
		detail = ?,
		provider_id = ?,
		priority = ?,
		target = ?,

		tm_update = ?
	where
		id = ?
	`

	ts := h.utilHandler.TimeGetCurTime()
	if _, err := h.db.Exec(q, name, detail, providerID.Bytes(), priority, target, ts, id.Bytes()); err != nil {
		return fmt.Errorf("could not execute the query. RouteUpdate. err: %v", err)
	}

	_ = h.routeUpdateToCache(ctx, id)

	return nil
}
