package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-route-manager/models/route"
)

const (
	routesTable = "route_routes"
)

// routeGetFromRow gets the route from the row.
func (h *handler) routeGetFromRow(row *sql.Rows) (*route.Route, error) {
	res := &route.Route{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. routeGetFromRow. err: %v", err)
	}

	return res, nil
}

// RouteCreate creates a new route record
func (h *handler) RouteCreate(ctx context.Context, r *route.Route) error {
	now := h.utilHandler.TimeGetCurTime()

	// Set timestamps
	r.TMCreate = now
	r.TMUpdate = commondatabasehandler.DefaultTimeStamp
	r.TMDelete = commondatabasehandler.DefaultTimeStamp

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(r)
	if err != nil {
		return fmt.Errorf("could not prepare fields. RouteCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(routesTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. RouteCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. RouteCreate. err: %v", err)
	}

	_ = h.routeUpdateToCache(ctx, r.ID)

	return nil
}

// routeUpdateToCache gets the route from the DB and update the cache.
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

// routeSetToCache sets the given route to the cache
func (h *handler) routeSetToCache(ctx context.Context, f *route.Route) error {
	if err := h.cache.RouteSet(ctx, f); err != nil {
		return err
	}

	return nil
}

// routeGetFromCache returns route from the cache if possible.
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
	fields := commondatabasehandler.GetDBFields(&route.Route{})
	query, args, err := squirrel.
		Select(fields...).
		From(routesTable).
		Where(squirrel.Eq{string(route.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. routeGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. routeGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. routeGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.routeGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. routeGetFromDB. id: %s, err: %v", id, err)
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
func (h *handler) RouteGets(ctx context.Context, token string, limit uint64, filters map[route.Field]any) ([]*route.Route, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&route.Route{})
	sb := squirrel.
		Select(fields...).
		From(routesTable).
		Where(squirrel.GtOrEq{string(route.FieldTMDelete): commondatabasehandler.DefaultTimeStamp}).
		Where(squirrel.Lt{string(route.FieldTMCreate): token}).
		OrderBy(string(route.FieldTMCreate) + " DESC", string(route.FieldID) + " DESC").
		Limit(limit).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. RouteGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. RouteGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. RouteGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*route.Route{}
	for rows.Next() {
		u, err := h.routeGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. RouteGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. RouteGets. err: %v", err)
	}

	return res, nil
}

// RouteDelete deletes the given route
func (h *handler) RouteDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeGetCurTime()

	fields := map[route.Field]any{
		route.FieldTMUpdate: ts,
		route.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("RouteDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(routesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(route.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("RouteDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("RouteDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %v", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	_ = h.cache.RouteDelete(ctx, id)

	return nil
}

// RouteUpdate updates the route information.
func (h *handler) RouteUpdate(ctx context.Context, id uuid.UUID, fields map[route.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[route.FieldTMUpdate] = h.utilHandler.TimeGetCurTime()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("RouteUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(routesTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(route.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("RouteUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("RouteUpdate: exec failed: %w", err)
	}

	_ = h.routeUpdateToCache(ctx, id)
	return nil
}
