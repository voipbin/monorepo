package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-webchat-manager/models/widget"
)

const (
	webchatWidgetsTable = "webchat_widgets"
)

// widgetGetFromRow gets the widget from the row.
func (h *handler) widgetGetFromRow(row *sql.Rows) (*widget.Widget, error) {
	res := &widget.Widget{}

	if err := commondatabasehandler.ScanRow(row, res); err != nil {
		return nil, fmt.Errorf("could not scan the row. widgetGetFromRow. err: %v", err)
	}

	return res, nil
}

// WidgetCreate creates new widget record.
func (h *handler) WidgetCreate(ctx context.Context, w *widget.Widget) error {
	now := h.utilHandler.TimeNow()

	// Set timestamps
	w.TMCreate = now
	w.TMUpdate = nil
	w.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(w)
	if err != nil {
		return fmt.Errorf("could not prepare fields. WidgetCreate. err: %v", err)
	}

	sb := squirrel.
		Insert(webchatWidgetsTable).
		SetMap(fields).
		PlaceholderFormat(squirrel.Question)

	query, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("could not build query. WidgetCreate. err: %v", err)
	}

	if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("could not execute query. WidgetCreate. err: %v", err)
	}

	// update the cache
	_ = h.widgetUpdateToCache(ctx, w.ID)

	return nil
}

// widgetUpdateToCache gets the widget from the DB and updates the cache.
func (h *handler) widgetUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.widgetGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.widgetSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// widgetSetToCache sets the given widget to the cache
func (h *handler) widgetSetToCache(ctx context.Context, w *widget.Widget) error {
	if err := h.cache.WidgetSet(ctx, w); err != nil {
		return err
	}

	return nil
}

// widgetGetFromCache returns widget from the cache.
func (h *handler) widgetGetFromCache(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	res, err := h.cache.WidgetGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// widgetGetFromDB returns widget from the DB.
func (h *handler) widgetGetFromDB(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	fields := commondatabasehandler.GetDBFields(&widget.Widget{})
	query, args, err := squirrel.
		Select(fields...).
		From(webchatWidgetsTable).
		Where(squirrel.Eq{string(widget.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build sql. widgetGetFromDB. err: %v", err)
	}

	row, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. widgetGetFromDB. err: %v", err)
	}
	defer func() {
		_ = row.Close()
	}()

	if !row.Next() {
		if err := row.Err(); err != nil {
			return nil, fmt.Errorf("row iteration error. widgetGetFromDB. err: %v", err)
		}
		return nil, ErrNotFound
	}

	res, err := h.widgetGetFromRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not get data from row. widgetGetFromDB. id: %s, err: %v", id, err)
	}

	return res, nil
}

// WidgetGet get widget from the database.
func (h *handler) WidgetGet(ctx context.Context, id uuid.UUID) (*widget.Widget, error) {
	res, err := h.widgetGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.widgetGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.widgetSetToCache(ctx, res)

	return res, nil
}

// WidgetList returns widgets.
func (h *handler) WidgetList(ctx context.Context, size uint64, token string, filters map[widget.Field]any) ([]*widget.Widget, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	fields := commondatabasehandler.GetDBFields(&widget.Widget{})
	sb := squirrel.
		Select(fields...).
		From(webchatWidgetsTable).
		Where(squirrel.Lt{string(widget.FieldTMCreate): token}).
		OrderBy(string(widget.FieldTMCreate) + " DESC").
		Limit(size).
		PlaceholderFormat(squirrel.Question)

	sb, err := commondatabasehandler.ApplyFields(sb, filters)
	if err != nil {
		return nil, fmt.Errorf("could not apply filters. WidgetList. err: %v", err)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query. WidgetList. err: %v", err)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query. WidgetList. err: %v", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	res := []*widget.Widget{}
	for rows.Next() {
		u, err := h.widgetGetFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("could not get data. WidgetList, err: %v", err)
		}
		res = append(res, u)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error. WidgetList. err: %v", err)
	}

	return res, nil
}

// WidgetUpdate updates widget fields.
func (h *handler) WidgetUpdate(ctx context.Context, id uuid.UUID, fields map[widget.Field]any) error {
	if len(fields) == 0 {
		return nil
	}

	fields[widget.FieldTMUpdate] = h.utilHandler.TimeNow()

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("WidgetUpdate: prepare fields failed: %w", err)
	}

	q := squirrel.Update(webchatWidgetsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(widget.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("WidgetUpdate: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("WidgetUpdate: exec failed: %w", err)
	}

	_ = h.widgetUpdateToCache(ctx, id)
	return nil
}

// WidgetDelete soft-deletes the widget.
func (h *handler) WidgetDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	fields := map[widget.Field]any{
		widget.FieldTMUpdate: ts,
		widget.FieldTMDelete: ts,
	}

	tmpFields, err := commondatabasehandler.PrepareFields(fields)
	if err != nil {
		return fmt.Errorf("WidgetDelete: prepare fields failed: %w", err)
	}

	sb := squirrel.Update(webchatWidgetsTable).
		SetMap(tmpFields).
		Where(squirrel.Eq{string(widget.FieldID): id.Bytes()}).
		PlaceholderFormat(squirrel.Question)

	sqlStr, args, err := sb.ToSql()
	if err != nil {
		return fmt.Errorf("WidgetDelete: build SQL failed: %w", err)
	}

	if _, err := h.db.ExecContext(ctx, sqlStr, args...); err != nil {
		return fmt.Errorf("WidgetDelete: exec failed: %w", err)
	}

	// update the cache
	_ = h.widgetUpdateToCache(ctx, id)

	return nil
}
