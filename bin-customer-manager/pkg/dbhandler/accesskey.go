package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/metricshandler"
)

const (
	accesskeyTable = "customer_accesskeys"
)

// accesskeyGetFromRow gets the accesskey from the row.
func (h *handler) accesskeyGetFromRow(row *sql.Rows) (*accesskey.Accesskey, error) {
	res := &accesskey.Accesskey{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. accesskeyGetFromRow. err: %v", err)
	}

	return res, nil
}

// AccesskeyCreate creates new accesskey record and returns the created accesskey record.
func (h *handler) AccesskeyCreate(ctx context.Context, c *accesskey.Accesskey) error {
	start := time.Now()
	status := "error"
	defer func() {
		metricshandler.DBOperationTotal.WithLabelValues("create", "accesskey", status).Inc()
		metricshandler.DBOperationDuration.WithLabelValues("create", "accesskey").Observe(float64(time.Since(start).Milliseconds()))
	}()

	now := h.utilHandler.TimeNow()

	// Set timestamps
	c.TMCreate = now
	c.TMUpdate = nil
	c.TMDelete = nil

	// Use PrepareFields to get field map
	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("could not prepare fields. AccesskeyCreate. err: %v", err)
	}

	// Use SetMap instead of Columns/Values
	sb := squirrel.
		Insert(accesskeyTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. AccesskeyCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. AccesskeyCreate. err: %v", err)
	}

	status = "success"

	// update the cache
	_ = h.accesskeyUpdateToCache(ctx, c.ID)

	return nil
}

// accesskeyUpdateToCache gets the accesskey from the DB and update the cache.
func (h *handler) accesskeyUpdateToCache(ctx context.Context, id uuid.UUID) error {

	res, err := h.accesskeyGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.accesskeySetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// accesskeySetToCache sets the given accesskey to the cache
func (h *handler) accesskeySetToCache(ctx context.Context, u *accesskey.Accesskey) error {
	if err := h.cache.AccesskeySet(ctx, u); err != nil {
		return err
	}

	return nil
}

// accesskeyGetFromCache returns accesskey from the cache.
func (h *handler) accesskeyGetFromCache(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {

	// get from cache
	res, err := h.cache.AccesskeyGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// accesskeyGetFromDB returns accesskey from the DB.
func (h *handler) accesskeyGetFromDB(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {
	fields := commondatabasehandler.GetDBFields(&accesskey.Accesskey{})
	query, args, err := squirrel.
		Select(fields...).
		From(accesskeyTable).
		Where(squirrel.Eq{string(accesskey.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. accesskeyGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. accesskeyGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. accesskeyGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.accesskeyGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. accesskeyGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// AccesskeyGet returns accesskey.
func (h *handler) AccesskeyGet(ctx context.Context, id uuid.UUID) (*accesskey.Accesskey, error) {
	res, err := h.accesskeyGetFromCache(ctx, id)
	if err == nil {
		metricshandler.CacheOperationTotal.WithLabelValues("get", "accesskey", "hit").Inc()
		return res, nil
	}
	metricshandler.CacheOperationTotal.WithLabelValues("get", "accesskey", "miss").Inc()

	start := time.Now()
	status := "error"
	defer func() {
		metricshandler.DBOperationTotal.WithLabelValues("get", "accesskey", status).Inc()
		metricshandler.DBOperationDuration.WithLabelValues("get", "accesskey").Observe(float64(time.Since(start).Milliseconds()))
	}()

	res, err = h.accesskeyGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	status = "success"

	// set to the cache
	_ = h.accesskeySetToCache(ctx, res)

	return res, nil
}

// AccesskeyGets returns accesskeys.
func (h *handler) AccesskeyList(ctx context.Context, size uint64, token string, filters map[accesskey.Field]any) ([]*accesskey.Accesskey, error) {
	start := time.Now()
	status := "error"
	defer func() {
		metricshandler.DBOperationTotal.WithLabelValues("list", "accesskey", status).Inc()
		metricshandler.DBOperationDuration.WithLabelValues("list", "accesskey").Observe(float64(time.Since(start).Milliseconds()))
	}()

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&accesskey.Accesskey{})
	sb := squirrel.
		Select(fields...).
		From(accesskeyTable).
		Where(squirrel.Lt{string(accesskey.FieldTMCreate): token}).
		OrderBy(string(accesskey.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. AccesskeyGets. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. AccesskeyGets. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. AccesskeyGets. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*accesskey.Accesskey{}
	for rows.Next() {
		u, err := h.accesskeyGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. AccesskeyGets, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. AccesskeyGets. err: %v", err)
	}

	status = "success"

	return res, nil
}

// AccesskeyUpdate updates accesskey fields.
func (h *handler) AccesskeyUpdate(ctx context.Context, id uuid.UUID, fields map[accesskey.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	start := time.Now()
	status := "error"
	defer func() {
		metricshandler.DBOperationTotal.WithLabelValues("update", "accesskey", status).Inc()
		metricshandler.DBOperationDuration.WithLabelValues("update", "accesskey").Observe(float64(time.Since(start).Milliseconds()))
	}()

	fields[accesskey.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("AccesskeyUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(accesskeyTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(accesskey.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("AccesskeyUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("AccesskeyUpdate: exec failed: %w", err)
	}

	status = "success"

	_ = h.accesskeyUpdateToCache(ctx, id)
	return nil
}

// AccesskeyDelete deletes the accesskey.
func (h *handler) AccesskeyDelete(ctx context.Context, id uuid.UUID) error {
	start := time.Now()
	status := "error"
	defer func() {
		metricshandler.DBOperationTotal.WithLabelValues("delete", "accesskey", status).Inc()
		metricshandler.DBOperationDuration.WithLabelValues("delete", "accesskey").Observe(float64(time.Since(start).Milliseconds()))
	}()

	ts := h.utilHandler.TimeNow()

	fields := map[accesskey.Field]any{
		accesskey.FieldTMUpdate: ts,
		accesskey.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("AccesskeyDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(accesskeyTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(accesskey.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("AccesskeyDelete: build SQL failed: %w", err)
	}

	result, err := h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("AccesskeyDelete: exec failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("could not get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return ErrNotFound
	}

	status = "success"

	// update the cache
	_ = h.accesskeyUpdateToCache(ctx, id)

	return nil
}
