package dbhandler

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/rag"
)

const tableRagRags = "rag_rags"

// ragScanRow scans a single row into a rag.Rag struct.
// gofrs/uuid implements sql.Scanner, so PostgreSQL UUID columns scan directly.
func ragScanRow(row *sql.Row) (*rag.Rag, error) {
	var r rag.Rag

	err := row.Scan(
		&r.ID,
		&r.CustomerID,
		&r.Name,
		&r.Description,
		&r.TMCreate,
		&r.TMUpdate,
		&r.TMDelete,
	)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// ragScanRows scans multiple rows into rag.Rag structs.
func ragScanRows(rows *sql.Rows) ([]*rag.Rag, error) {
	res := []*rag.Rag{}

	for rows.Next() {
		var r rag.Rag

		err := rows.Scan(
			&r.ID,
			&r.CustomerID,
			&r.Name,
			&r.Description,
			&r.TMCreate,
			&r.TMUpdate,
			&r.TMDelete,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

// ragColumns returns the column names for the rag_rags table.
func ragColumns() []string {
	return []string{
		"id",
		"customer_id",
		"name",
		"description",
		"tm_create",
		"tm_update",
		"tm_delete",
	}
}

// RagCreate inserts a new rag record.
// Timestamps are set in Go so the caller's struct is populated after insert.
func (h *handler) RagCreate(ctx context.Context, r *rag.Rag) error {
	r.TMCreate = h.utilHandler.TimeNow()
	r.TMUpdate = h.utilHandler.TimeNow()

	q := psql.
		Insert(tableRagRags).
		Columns(
			"id",
			"customer_id",
			"name",
			"description",
			"tm_create",
			"tm_update",
		).
		Values(
			r.ID,
			r.CustomerID,
			r.Name,
			r.Description,
			r.TMCreate,
			r.TMUpdate,
		)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build rag insert query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute rag insert: %w", err)
	}

	return nil
}

// RagGet retrieves a rag by ID
func (h *handler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	q := psql.
		Select(ragColumns()...).
		From(tableRagRags).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build rag get query: %w", err)
	}

	row := h.db.QueryRowContext(ctx, sqlStr, args...)

	r, err := ragScanRow(row)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("could not scan rag row: %w", err)
	}

	return r, nil
}

// RagList retrieves rags matching the given filters with cursor-based pagination.
// Pagination uses tm_create as cursor (token). Filters are applied as WHERE clauses.
// The "deleted" filter controls tm_delete: false = IS NULL, true = IS NOT NULL.
func (h *handler) RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}
	if size == 0 {
		size = 100
	}

	q := psql.
		Select(ragColumns()...).
		From(tableRagRags).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create DESC").
		Limit(size)

	hasDeletedFilter := false
	for k, v := range filters {
		key := string(k)
		switch key {
		case "deleted":
			hasDeletedFilter = true
			deleted, ok := v.(bool)
			if ok && !deleted {
				q = q.Where("tm_delete IS NULL")
			} else if ok && deleted {
				q = q.Where("tm_delete IS NOT NULL")
			}
		default:
			q = q.Where(sq.Eq{key: v})
		}
	}
	if !hasDeletedFilter {
		q = q.Where("tm_delete IS NULL")
	}

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build rag list query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute rag list query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return ragScanRows(rows)
}

// RagUpdate updates rag fields by ID
func (h *handler) RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) error {
	updateMap := map[string]any{
		"tm_update": h.utilHandler.TimeNow(),
	}
	for k, v := range fields {
		updateMap[string(k)] = v
	}

	q := psql.
		Update(tableRagRags).
		SetMap(updateMap).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build rag update query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute rag update: %w", err)
	}

	return nil
}

// RagDelete soft-deletes a rag by ID
func (h *handler) RagDelete(ctx context.Context, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()

	q := psql.
		Update(tableRagRags).
		SetMap(map[string]any{
			"tm_delete": now,
			"tm_update": now,
		}).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build rag delete query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute rag soft delete: %w", err)
	}

	return nil
}
