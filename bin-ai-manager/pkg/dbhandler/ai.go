package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	uuid "github.com/gofrs/uuid"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-ai-manager/models/ai"
)

const (
	aiTable = "ai_ais"
)

// ErrNotInsightAI is returned by AIActivateInsight when the target AI is not
// a type=insight AI. Activation is meaningless for any other type.
var ErrNotInsightAI = fmt.Errorf("ai is not an insight ai")

// AICreate creates new ai record.
func (h *handler) AICreate(ctx context.Context, c *ai.AI) error {
	c.TMCreate = h.utilHandler.TimeNow()
	c.TMUpdate = nil
	c.TMDelete = nil

	fields, err := commondatabasehandler.PrepareFields(c)
	if err != nil {
		return fmt.Errorf("AICreate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Insert(aiTable).SetMap(fields).ToSql()
	if err != nil {
		return fmt.Errorf("AICreate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AICreate: could not execute query. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, c.ID)

	return nil
}

// aiGetFromCache returns ai from the cache.
func (h *handler) aiGetFromCache(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, err := h.cache.AIGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// aiGetFromDB returns ai from the DB.
func (h *handler) aiGetFromDB(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	cols := commondatabasehandler.GetDBFields(ai.AI{})

	query, args, err := sq.Select(cols...).
		From(aiTable).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("aiGetFromDB: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("aiGetFromDB: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, ErrNotFound
	}

	res := &ai.AI{}
	if err := commondatabasehandler.ScanRow(rows, res); err != nil {
		return nil, fmt.Errorf("aiGetFromDB: could not scan row. err: %v", err)
	}

	return res, nil
}

// aiUpdateToCache gets the ai from the DB and updates the cache.
func (h *handler) aiUpdateToCache(ctx context.Context, id uuid.UUID) error {
	res, err := h.aiGetFromDB(ctx, id)
	if err != nil {
		return err
	}

	if err := h.aiSetToCache(ctx, res); err != nil {
		return err
	}

	return nil
}

// aiSetToCache sets the given ai to the cache.
func (h *handler) aiSetToCache(ctx context.Context, c *ai.AI) error {
	if err := h.cache.AISet(ctx, c); err != nil {
		return err
	}

	return nil
}

// AIGet returns ai.
func (h *handler) AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error) {
	res, err := h.aiGetFromCache(ctx, id)
	if err == nil {
		return res, nil
	}

	res, err = h.aiGetFromDB(ctx, id)
	if err != nil {
		return nil, err
	}

	// set to the cache
	_ = h.aiSetToCache(ctx, res)

	return res, nil
}

// AIDelete deletes the ai.
func (h *handler) AIDelete(ctx context.Context, id uuid.UUID) error {
	ts := h.utilHandler.TimeNow()

	query, args, err := sq.Update(aiTable).
		SetMap(map[string]any{
			"tm_update": ts,
			"tm_delete": ts,
			// Cleared unconditionally, never "only if it was true". A read-then-write
			// would race a concurrent AIActivateInsight and could silently undo a
			// just-won activation; clearing blindly is a harmless no-op otherwise.
			// Required because the generated column's tm_delete IS NULL guard frees
			// the unique-index slot on soft delete, which would otherwise leave a
			// stale is_insight_active = true readable on a deleted row.
			"is_insight_active": false,
		}).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIDelete: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIDelete: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, id)

	return nil
}

// AIGets returns a list of ais.
func (h *handler) AIList(ctx context.Context, size uint64, token string, filters map[ai.Field]any) ([]*ai.AI, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	cols := commondatabasehandler.GetDBFields(ai.AI{})

	builder := sq.Select(cols...).
		From(aiTable).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create desc").
		Limit(size)

	builder, err := commondatabasehandler.ApplyFields(builder, filters)
	if err != nil {
		return nil, fmt.Errorf("AIList: could not apply filters. err: %v", err)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AIList: could not build query. err: %v", err)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("AIList: could not query. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	res := []*ai.AI{}
	for rows.Next() {
		u := &ai.AI{}
		if err := commondatabasehandler.ScanRow(rows, u); err != nil {
			return nil, fmt.Errorf("AIList: could not scan row. err: %v", err)
		}
		res = append(res, u)
	}

	return res, nil
}

// AIActivateInsight makes the given type=insight AI the customer's single
// active Insight AI, clearing whichever one was active before. Both rows are
// mutated in one transaction so the ai_ais.active_insight_key unique index is
// never transiently violated.
//
// Idempotent: activating the already-active AI succeeds and changes nothing
// observable -- no row is written, so tm_update is not bumped and no previous
// AI is reported back to the caller.
//
// Returns the activated AI and, when the activation actually displaced a
// different AI, the row that was just deactivated (nil otherwise). The caller
// needs the deactivated row to publish its own update event: external
// subscribers would otherwise keep a stale is_insight_active=true for it.
//
// Returns ErrNotFound if the target does not exist or is soft-deleted, and
// ErrNotInsightAI if the target is not type=insight. A concurrent activation
// for the same customer surfaces as a duplicate-key error (see IsErrDuplicate).
//
// Locking: the FOR UPDATE below covers the customer's currently-active insight
// row, not the target row. Widening it is unnecessary for correctness -- a race
// with a concurrent "change target type to normal" could leave
// is_insight_active=true on a type=normal row, but the generated column's
// type='insight' guard means such a row never holds the unique-index slot and
// the Case-panel resolution query filters type=insight, so it is never selected.
// The analogous race with a concurrent AIDelete (the target is soft-deleted
// between the step-1 read and the step-4 write) is benign for the same reason:
// the generated column's tm_delete guard keeps a deleted row out of the unique
// index, and every reader filters tm_delete IS NULL, so a leftover
// is_insight_active=true on a deleted row is never observed.
func (h *handler) AIActivateInsight(ctx context.Context, id uuid.UUID) (*ai.AI, *ai.AI, error) {
	var (
		err        error
		committed  bool
		previousID uuid.UUID
	)

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("AIActivateInsight: BeginTx: %w", err)
	}
	defer func() {
		if !committed {
			_ = tx.Rollback()
			return
		}
		// Cache refresh happens only after a successful Commit, and covers BOTH
		// mutated rows -- AIGet is cache-first, so skipping either would leave the
		// Case panel resolving a stale AI indefinitely. context.Background() so a
		// cancelled request context cannot skip it.
		if previousID != uuid.Nil && previousID != id {
			_ = h.aiUpdateToCache(context.Background(), previousID)
		}
		_ = h.aiUpdateToCache(context.Background(), id)
	}()

	// 1. Read the target row to derive the customer scope and validate the type.
	// A soft-deleted target is filtered out here rather than reported
	// separately: an activated-but-deleted AI is not a distinguishable state a
	// caller can act on, so both collapse to ErrNotFound.
	var targetCustomer []byte
	var targetType string
	err = tx.QueryRowContext(ctx, `
		SELECT customer_id, type
		FROM `+aiTable+`
		WHERE id = ? AND tm_delete IS NULL
	`, id.Bytes()).Scan(&targetCustomer, &targetType)
	if err == sql.ErrNoRows {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("AIActivateInsight: select target: %w", err)
	}
	if targetType != string(ai.TypeInsight) {
		return nil, nil, ErrNotInsightAI
	}

	customerID, err := uuid.FromBytes(targetCustomer)
	if err != nil {
		return nil, nil, fmt.Errorf("AIActivateInsight: parse customer id: %w", err)
	}

	// 2. Lock the customer's currently-active insight row, if any.
	var activeIDBytes []byte
	errActive := tx.QueryRowContext(ctx, `
		SELECT id
		FROM `+aiTable+`
		WHERE customer_id = ? AND type = ? AND tm_delete IS NULL AND is_insight_active = TRUE
		FOR UPDATE
	`, customerID.Bytes(), string(ai.TypeInsight)).Scan(&activeIDBytes)
	switch {
	case errActive == sql.ErrNoRows:
		// No active insight AI yet -- nothing to clear.
	case errActive != nil:
		return nil, nil, fmt.Errorf("AIActivateInsight: select active: %w", errActive)
	default:
		previousID, err = uuid.FromBytes(activeIDBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("AIActivateInsight: parse active id: %w", err)
		}
	}

	// The target is already the active one: steps 3 and 4 are both no-ops.
	// Writing anyway would bump tm_update and emit an update event for a state
	// that did not change, so re-activation stays a true no-op instead.
	alreadyActive := previousID == id

	now := h.utilHandler.TimeNow()

	// 3. Clear the previously-active row (skipped when it is the target itself).
	if previousID != uuid.Nil && !alreadyActive {
		if _, err = tx.ExecContext(ctx, `
			UPDATE `+aiTable+`
			SET is_insight_active = FALSE, tm_update = ?
			WHERE id = ?
		`, now, previousID.Bytes()); err != nil {
			return nil, nil, fmt.Errorf("AIActivateInsight: clear previous: %w", err)
		}
	}

	// 4. Activate the target row (skipped when it is already TRUE).
	if !alreadyActive {
		if _, err = tx.ExecContext(ctx, `
			UPDATE `+aiTable+`
			SET is_insight_active = TRUE, tm_update = ?
			WHERE id = ?
		`, now, id.Bytes()); err != nil {
			return nil, nil, fmt.Errorf("AIActivateInsight: activate target: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("AIActivateInsight: commit: %w", err)
	}
	committed = true

	// Read straight from the DB: the deferred cache refresh has not run yet, so
	// a cache-first AIGet here could still return the pre-activation row.
	res, err := h.aiGetFromDB(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("AIActivateInsight: could not get activated ai. err: %w", err)
	}

	// The deactivated row is returned so the caller can publish its update
	// event; nil when nothing was displaced.
	var previous *ai.AI
	if previousID != uuid.Nil && !alreadyActive {
		previous, err = h.aiGetFromDB(ctx, previousID)
		if err != nil {
			return nil, nil, fmt.Errorf("AIActivateInsight: could not get deactivated ai. err: %w", err)
		}
	}

	return res, previous, nil
}

// AIUpdate updates the ai fields.
func (h *handler) AIUpdate(ctx context.Context, id uuid.UUID, fields map[ai.Field]any) error {
	updateFields := make(map[string]any)
	for k, v := range fields {
		updateFields[string(k)] = v
	}
	updateFields["tm_update"] = h.utilHandler.TimeNow()

	preparedFields, err := commondatabasehandler.PrepareFields(updateFields)
	if err != nil {
		return fmt.Errorf("AIUpdate: could not prepare fields. err: %v", err)
	}

	query, args, err := sq.Update(aiTable).
		SetMap(preparedFields).
		Where(sq.Eq{"id": id.Bytes()}).
		ToSql()
	if err != nil {
		return fmt.Errorf("AIUpdate: could not build query. err: %v", err)
	}

	_, err = h.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("AIUpdate: could not execute. err: %v", err)
	}

	// update the cache
	_ = h.aiUpdateToCache(ctx, id)

	return nil
}
