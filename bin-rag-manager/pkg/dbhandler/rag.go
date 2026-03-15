package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/rag"
)

const tableRagRags = "rag_rags"

// ragScanRow scans a single row into a rag.Rag struct.
func ragScanRow(row *sql.Row) (*rag.Rag, error) {
	var r rag.Rag
	var idBytes, customerIDBytes []byte

	err := row.Scan(
		&idBytes,
		&customerIDBytes,
		&r.Name,
		&r.Description,
		&r.TMCreate,
		&r.TMUpdate,
		&r.TMDelete,
	)
	if err != nil {
		return nil, err
	}

	r.ID, _ = uuid.FromBytes(idBytes)
	r.CustomerID, _ = uuid.FromBytes(customerIDBytes)

	return &r, nil
}

// ragScanRows scans multiple rows into rag.Rag structs.
func ragScanRows(rows *sql.Rows) ([]*rag.Rag, error) {
	res := []*rag.Rag{}

	for rows.Next() {
		var r rag.Rag
		var idBytes, customerIDBytes []byte

		err := rows.Scan(
			&idBytes,
			&customerIDBytes,
			&r.Name,
			&r.Description,
			&r.TMCreate,
			&r.TMUpdate,
			&r.TMDelete,
		)
		if err != nil {
			return nil, err
		}

		r.ID, _ = uuid.FromBytes(idBytes)
		r.CustomerID, _ = uuid.FromBytes(customerIDBytes)

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

// RagCreate inserts a new rag record
func (h *handler) RagCreate(ctx context.Context, r *rag.Rag) error {
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
			r.ID.Bytes(),
			r.CustomerID.Bytes(),
			r.Name,
			r.Description,
			sq.Expr("NOW()"),
			sq.Expr("NOW()"),
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
		Where(sq.Eq{"id": id.Bytes()}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build rag get query: %w", err)
	}

	row := h.db.QueryRowContext(ctx, sqlStr, args...)

	r, err := ragScanRow(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan rag row: %w", err)
	}

	return r, nil
}

// RagGetsByCustomerID retrieves all rags for a customer
func (h *handler) RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error) {
	q := psql.
		Select(ragColumns()...).
		From(tableRagRags).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build rag gets query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute rag gets query: %w", err)
	}
	defer rows.Close()

	res, err := ragScanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan rag rows: %w", err)
	}

	return res, nil
}

// RagUpdate updates rag fields by ID
func (h *handler) RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) error {
	updateMap := map[string]any{
		"tm_update": sq.Expr("NOW()"),
	}
	for k, v := range fields {
		updateMap[string(k)] = v
	}

	q := psql.
		Update(tableRagRags).
		SetMap(updateMap).
		Where(sq.Eq{"id": id.Bytes()}).
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
	q := psql.
		Update(tableRagRags).
		SetMap(map[string]any{
			"tm_delete": sq.Expr("NOW()"),
			"tm_update": sq.Expr("NOW()"),
		}).
		Where(sq.Eq{"id": id.Bytes()}).
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
