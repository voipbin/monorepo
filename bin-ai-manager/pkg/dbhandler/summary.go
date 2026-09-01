package dbhandler

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/summary"
)

const (
	summaryTable = "ai_summaries"
)

// SummaryCreate creates a new summary record.
func (h *handler) SummaryCreate(ctx context.Context, c *summary.Summary) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = nil
	c.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("SummaryCreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(summaryTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("SummaryCreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("SummaryCreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.summaryUpdateToCache(ctx, c.ID)

	return nil
}

// summaryGetFromCache returns summary from the cache.
func (h *handler) summaryGetFromCache(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {
	res, err := h.cache.SummaryGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// summaryGetFromDB returns summary from the DB.
func (h *handler) summaryGetFromDB(id uuid.UUID) (*summary.Summary, error) {
	cols := commondatabasehandler.GetDBFields(summary.Summary{})

	query, args, err := sq.Select(cols...).
		From(summaryTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("summaryGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("summaryGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &summary.Summary{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("summaryGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// summaryUpdateToCache gets the summary from the DB and updates the cache.
func (h *handler) summaryUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.summaryGetFromDB(id)
	if err != nil {
		return err
	}

	if err := h.summarySetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// summarySetToCache sets the given summary to the cache.
func (h *handler) summarySetToCache(ctx context.Context, c *summary.Summary) error {
	if err := h.cache.SummarySet(ctx, c); err != nil {
		return err
	}

	return nil
}

// SummaryGet returns summary.
func (h *handler) SummaryGet(ctx context.Context, id uuid.UUID) (*summary.Summary, error) {
	res, err := h.summaryGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.summaryGetFromDB(id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.summarySetToCache(ctx, res)

	return res, nil
}

// SummaryDelete deletes the summary.
func (h *handler) SummaryDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(summaryTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("SummaryDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("SummaryDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.summaryUpdateToCache(ctx, id)

	return nil
}

// SummaryGets returns a list of summaries.
func (h *handler) SummaryList(ctx context.Context, size uint64, token string, filters map[summary.Field]any) ([]*summary.Summary, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(summary.Summary{})

	builder := sq.Select(cols...).
		From(summaryTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("SummaryList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("SummaryList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("SummaryList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*summary.Summary{}
	for rows.Next() {
		u := &summary.Summary{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("SummaryList: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// SummaryUpdate updates the summary fields unconditionally (no rows-affected check).
// As of VOIP-1422, its only production caller (summaryhandler.UpdateStatusDone) was
// rewired to the conditional SummaryUpdateStatusDoneIfNotDone below, since an
// unconditional update cannot protect against bin-conference-manager's double
// conference_deleted delivery. Retained as general-purpose infrastructure for any
// future non-status field update that does not need that guard; currently unreferenced
// outside its own test.
func (h *handler) SummaryUpdate(ctx context.Context, id uuid.UUID, fields map[summary.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("SummaryUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(summaryTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("SummaryUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("SummaryUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.summaryUpdateToCache(ctx, id)

	return nil
}

// SummaryUpdateStatusDoneIfNotDone conditionally updates the summary's status to
// done and sets its content, but ONLY if the summary is not already done (VOIP-1422:
// bin-conference-manager can publish conference_deleted twice for one conference --
// Delete()'s own publish, then Destroy()'s once the asynchronously-kicked participants
// actually leave -- and neither ContentProcess nor its downstream, including
// startOnEndFlow, is safe to run twice). Returns the number of rows actually updated:
// 0 means the summary was already done when this write was attempted and the caller
// MUST NOT treat this as a fresh finalization (no cache refresh, no webhook, no
// on-end-flow trigger) -- matches the same conditional-update-plus-rows-affected
// pattern AIcallUpdateIfActive (pkg/dbhandler/aicall.go) already uses for the
// equivalent problem on AIcalls.
func (h *handler) SummaryUpdateStatusDoneIfNotDone(ctx context.Context, id uuid.UUID, content string) (int64, error) {
	updateFields := make(map[string]any)
	updateFields[string(summary.FieldStatus)] = summary.StatusDone
	updateFields[string(summary.FieldContent)] = content
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return 0, fmt.Errorf("SummaryUpdateStatusDoneIfNotDone: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(summaryTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		Where(sq.NotEq{"status": string(summary.StatusDone)}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("SummaryUpdateStatusDoneIfNotDone: could not build query. err: %v", err)
	}

	res, err := h.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("SummaryUpdateStatusDoneIfNotDone: could not execute. err: %v", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("SummaryUpdateStatusDoneIfNotDone: could not get rows affected. err: %v", err)
	}

	if n > 0 {
		// update the cache
		_ = h.summaryUpdateToCache(ctx, id)
	}

	return n, nil
}
