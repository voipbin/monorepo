package dbhandler

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
)

const tableDocuments = "rag_documents"

// documentColumns lists all columns in the rag_documents table in scan order.
var documentColumns = []string{
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
}

// psql is a squirrel StatementBuilder configured for PostgreSQL dollar placeholders.
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// scanDocument scans a single row into a Document struct.
// UUID columns are scanned as []byte and converted via uuid.FromBytes.
// Nullable columns (storage_file_id, source_url) use pointer types.
func scanDocument(row *sql.Row) (*document.Document, error) {
	var d document.Document

	var idBytes, customerIDBytes, ragIDBytes []byte
	var storageFileIDBytes *[]byte
	var sourceURL *string

	err := row.Scan(
		&idBytes,
		&customerIDBytes,
		&ragIDBytes,
		&d.Name,
		&d.DocType,
		&storageFileIDBytes,
		&sourceURL,
		&d.Status,
		&d.StatusMessage,
		&d.TMCreate,
		&d.TMUpdate,
		&d.TMDelete,
	)
	if err != nil {
		return nil, err
	}

	d.ID, _ = uuid.FromBytes(idBytes)
	d.CustomerID, _ = uuid.FromBytes(customerIDBytes)
	d.RagID, _ = uuid.FromBytes(ragIDBytes)

	if storageFileIDBytes != nil {
		d.StorageFileID, _ = uuid.FromBytes(*storageFileIDBytes)
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

		var idBytes, customerIDBytes, ragIDBytes []byte
		var storageFileIDBytes *[]byte
		var sourceURL *string

		err := rows.Scan(
			&idBytes,
			&customerIDBytes,
			&ragIDBytes,
			&d.Name,
			&d.DocType,
			&storageFileIDBytes,
			&sourceURL,
			&d.Status,
			&d.StatusMessage,
			&d.TMCreate,
			&d.TMUpdate,
			&d.TMDelete,
		)
		if err != nil {
			return nil, err
		}

		d.ID, _ = uuid.FromBytes(idBytes)
		d.CustomerID, _ = uuid.FromBytes(customerIDBytes)
		d.RagID, _ = uuid.FromBytes(ragIDBytes)

		if storageFileIDBytes != nil {
			d.StorageFileID, _ = uuid.FromBytes(*storageFileIDBytes)
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

// DocumentCreate inserts a new document record
func (h *handler) DocumentCreate(ctx context.Context, d *document.Document) error {
	// Handle nullable storage_file_id: use nil for zero UUID
	var storageFileID any
	if d.StorageFileID == uuid.Nil {
		storageFileID = nil
	} else {
		storageFileID = d.StorageFileID.Bytes()
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
			d.ID.Bytes(),
			d.CustomerID.Bytes(),
			d.RagID.Bytes(),
			d.Name,
			d.DocType,
			storageFileID,
			d.SourceURL,
			d.Status,
			d.StatusMessage,
			sq.Expr("NOW()"),
			sq.Expr("NOW()"),
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
		Select(documentColumns...).
		From(tableDocuments).
		Where(sq.Eq{"id": id.Bytes()}).
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

// DocumentGetsByRagID retrieves all documents for a rag
func (h *handler) DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error) {
	q := psql.
		Select(documentColumns...).
		From(tableDocuments).
		Where(sq.Eq{"rag_id": ragID.Bytes()}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build document gets by rag_id query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute document gets by rag_id query: %w", err)
	}
	defer rows.Close()

	res, err := scanDocumentRows(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan document rows: %w", err)
	}

	return res, nil
}

// DocumentGetsByCustomerID retrieves all documents for a customer
func (h *handler) DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error) {
	q := psql.
		Select(documentColumns...).
		From(tableDocuments).
		Where(sq.Eq{"customer_id": customerID.Bytes()}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build document gets by customer_id query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute document gets by customer_id query: %w", err)
	}
	defer rows.Close()

	res, err := scanDocumentRows(rows)
	if err != nil {
		return nil, fmt.Errorf("could not scan document rows: %w", err)
	}

	return res, nil
}

// DocumentUpdate updates document fields by ID
func (h *handler) DocumentUpdate(ctx context.Context, id uuid.UUID, fields map[document.Field]any) error {
	updateMap := map[string]any{
		"tm_update": sq.Expr("NOW()"),
	}
	for k, v := range fields {
		updateMap[string(k)] = v
	}

	q := psql.
		Update(tableDocuments).
		SetMap(updateMap).
		Where(sq.Eq{"id": id.Bytes()}).
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
	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_delete": sq.Expr("NOW()"),
			"tm_update": sq.Expr("NOW()"),
		}).
		Where(sq.Eq{"id": id.Bytes()}).
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
	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_delete": sq.Expr("NOW()"),
			"tm_update": sq.Expr("NOW()"),
		}).
		Where(sq.Eq{"rag_id": ragID.Bytes()}).
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
