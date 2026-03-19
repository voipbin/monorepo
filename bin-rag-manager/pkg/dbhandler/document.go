package dbhandler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
)

const tableDocuments = "rag_documents"

// documentColumns returns the column names for the rag_documents table in scan order.
func documentColumns() []string {
	return []string{
		"id",
		"customer_id",
		"rag_id",
		"name",
		"doc_type",
		"storage_file_id",
		"source_url",
		"status",
		"status_message",
		"tm_create",
		"tm_update",
		"tm_delete",
		"retry_count",
		"tm_processing",
	}
}

// scanDocument scans a single row into a Document struct.
// gofrs/uuid implements sql.Scanner, so PostgreSQL UUID columns scan directly.
// Nullable columns (storage_file_id, source_url) use uuid.NullUUID and *string.
func scanDocument(row *sql.Row) (*document.Document, error) {
	var d document.Document

	var storageFileID uuid.NullUUID
	var sourceURL *string

	err := row.Scan(
		&d.ID,
		&d.CustomerID,
		&d.RagID,
		&d.Name,
		&d.DocType,
		&storageFileID,
		&sourceURL,
		&d.Status,
		&d.StatusMessage,
		&d.TMCreate,
		&d.TMUpdate,
		&d.TMDelete,
		&d.RetryCount,
		&d.TMProcessing,
	)
	if err != nil {
		return nil, err
	}

	if storageFileID.Valid {
		d.StorageFileID = storageFileID.UUID
	}
	if sourceURL != nil {
		d.SourceURL = *sourceURL
	}

	return &d, nil
}

// scanDocumentRows scans multiple rows into a slice of Document structs.
func scanDocumentRows(rows *sql.Rows) ([]*document.Document, error) {
	res := []*document.Document{}

	for rows.Next() {
		var d document.Document

		var storageFileID uuid.NullUUID
		var sourceURL *string

		err := rows.Scan(
			&d.ID,
			&d.CustomerID,
			&d.RagID,
			&d.Name,
			&d.DocType,
			&storageFileID,
			&sourceURL,
			&d.Status,
			&d.StatusMessage,
			&d.TMCreate,
			&d.TMUpdate,
			&d.TMDelete,
			&d.RetryCount,
			&d.TMProcessing,
		)
		if err != nil {
			return nil, err
		}

		if storageFileID.Valid {
			d.StorageFileID = storageFileID.UUID
		}
		if sourceURL != nil {
			d.SourceURL = *sourceURL
		}

		res = append(res, &d)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

// DocumentCreate inserts a new document record.
// Timestamps are set in Go so the caller's struct is populated after insert.
func (h *handler) DocumentCreate(ctx context.Context, d *document.Document) error {
	d.TMCreate = h.utilHandler.TimeNow()
	d.TMUpdate = h.utilHandler.TimeNow()

	// Handle nullable storage_file_id: use nil for zero UUID
	var storageFileID any
	if d.StorageFileID == uuid.Nil {
		storageFileID = nil
	} else {
		storageFileID = d.StorageFileID
	}

	q := psql.
		Insert(tableDocuments).
		Columns(
			"id",
			"customer_id",
			"rag_id",
			"name",
			"doc_type",
			"storage_file_id",
			"source_url",
			"status",
			"status_message",
			"tm_create",
			"tm_update",
		).
		Values(
			d.ID,
			d.CustomerID,
			d.RagID,
			d.Name,
			d.DocType,
			storageFileID,
			d.SourceURL,
			d.Status,
			d.StatusMessage,
			d.TMCreate,
			d.TMUpdate,
		)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build document insert query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute document insert: %w", err)
	}

	return nil
}

// DocumentGet retrieves a document by ID
func (h *handler) DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build document get query: %w", err)
	}

	row := h.db.QueryRowContext(ctx, sqlStr, args...)
	d, err := scanDocument(row)
	if err != nil {
		return nil, fmt.Errorf("could not scan document: %w", err)
	}

	return d, nil
}

// DocumentList retrieves documents matching the given filters with cursor-based pagination.
func (h *handler) DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}
	if size == 0 {
		size = 100
	}

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
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
		return nil, fmt.Errorf("could not build document list query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute document list query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDocumentRows(rows)
}

// DocumentUpdate updates document fields by ID
func (h *handler) DocumentUpdate(ctx context.Context, id uuid.UUID, fields map[document.Field]any) error {
	updateMap := map[string]any{
		"tm_update": h.utilHandler.TimeNow(),
	}
	for k, v := range fields {
		updateMap[string(k)] = v
	}

	q := psql.
		Update(tableDocuments).
		SetMap(updateMap).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build document update query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute document update: %w", err)
	}

	return nil
}

// DocumentDelete soft-deletes a document by ID
func (h *handler) DocumentDelete(ctx context.Context, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_delete": now,
			"tm_update": now,
		}).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build document delete query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute document delete: %w", err)
	}

	return nil
}

// DocumentDeleteByRagID soft-deletes all documents for a rag
func (h *handler) DocumentDeleteByRagID(ctx context.Context, ragID uuid.UUID) error {
	now := h.utilHandler.TimeNow()

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_delete": now,
			"tm_update": now,
		}).
		Where(sq.Eq{"rag_id": ragID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build document delete by rag_id query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not execute document delete by rag_id: %w", err)
	}

	return nil
}

// DocumentClaimForProcessing atomically sets a pending document to processing status.
// It increments retry_count and sets tm_processing for heartbeat tracking.
// Returns the updated document or an error if the document is not in pending status.
func (h *handler) DocumentClaimForProcessing(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	now := h.utilHandler.TimeNow()

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"status":        document.StatusProcessing,
			"tm_processing": now,
			"tm_update":     now,
			"retry_count":   sq.Expr("retry_count + 1"),
		}).
		Where(sq.Eq{"id": id, "status": document.StatusPending}).
		Where("tm_delete IS NULL").
		Suffix("RETURNING " + strings.Join(documentColumns(), ", "))

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build claim query: %w", err)
	}

	row := h.db.QueryRowContext(ctx, sqlStr, args...)
	return scanDocument(row)
}

// DocumentUpdateHeartbeat updates the tm_processing timestamp for a document.
// Used by workers to signal they are still actively processing.
func (h *handler) DocumentUpdateHeartbeat(ctx context.Context, id uuid.UUID) error {
	now := h.utilHandler.TimeNow()

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_processing": now,
		}).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build heartbeat query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not update heartbeat: %w", err)
	}

	return nil
}

// DocumentGetStale retrieves documents stuck in processing state beyond the given threshold.
func (h *handler) DocumentGetStale(ctx context.Context, threshold time.Duration) ([]*document.Document, error) {
	cutoff := h.utilHandler.TimeNow().Add(-threshold)

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"status": document.StatusProcessing}).
		Where(sq.Lt{"tm_processing": cutoff}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build stale query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query stale documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDocumentRows(rows)
}

// DocumentGetPending retrieves documents in pending status with retry_count below the limit.
func (h *handler) DocumentGetPending(ctx context.Context) ([]*document.Document, error) {
	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"status": document.StatusPending}).
		Where(sq.Lt{"retry_count": 3}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build pending query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query pending documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDocumentRows(rows)
}

// DocumentResetStaleToPending resets documents stuck in processing back to pending status.
func (h *handler) DocumentResetStaleToPending(ctx context.Context, threshold time.Duration) error {
	now := h.utilHandler.TimeNow()
	cutoff := now.Add(-threshold)

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"status":    document.StatusPending,
			"tm_update": now,
		}).
		Where(sq.Eq{"status": document.StatusProcessing}).
		Where(sq.Lt{"tm_processing": cutoff}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("could not build reset query: %w", err)
	}

	_, err = h.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("could not reset stale documents: %w", err)
	}

	return nil
}

// DocumentGetsByRagID retrieves all non-deleted documents for a given rag ID.
func (h *handler) DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error) {
	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"rag_id": ragID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query documents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDocumentRows(rows)
}

// DocumentGetsByRagIDs retrieves all non-deleted documents for multiple rag IDs,
// grouped by rag ID in the returned map.
func (h *handler) DocumentGetsByRagIDs(ctx context.Context, ragIDs []uuid.UUID) (map[uuid.UUID][]*document.Document, error) {
	if len(ragIDs) == 0 {
		return map[uuid.UUID][]*document.Document{}, nil
	}

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"rag_id": ragIDs}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build batch query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query documents by rag IDs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	docs, err := scanDocumentRows(rows)
	if err != nil {
		return nil, err
	}

	res := map[uuid.UUID][]*document.Document{}
	for _, d := range docs {
		res[d.RagID] = append(res[d.RagID], d)
	}
	return res, nil
}
