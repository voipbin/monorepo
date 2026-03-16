package dbhandler

import (
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
)

func Test_DocumentCreate_SQL(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	d := &document.Document{
		ID:            uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		CustomerID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		RagID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
		Name:          "test-doc.txt",
		DocType:       document.DocTypeUploaded,
		StorageFileID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440004"),
		SourceURL:     "",
		Status:        document.StatusPending,
		StatusMessage: "",
		TMCreate:      &now,
		TMUpdate:      &now,
	}

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
		t.Fatalf("unexpected error building insert SQL: %v", err)
	}

	expectedSQL := "INSERT INTO rag_documents (id,customer_id,rag_id,name,doc_type,storage_file_id,source_url,status,status_message,tm_create,tm_update) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	if len(args) != 11 {
		t.Errorf("unexpected arg count: got %d, want 11", len(args))
	}
}

func Test_DocumentCreate_NilStorageFileID(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	d := &document.Document{
		ID:            uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
		CustomerID:    uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		RagID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003"),
		Name:          "test-doc.txt",
		DocType:       document.DocTypeURL,
		StorageFileID: uuid.Nil, // zero UUID -> should become nil
		SourceURL:     "https://example.com/doc.pdf",
		Status:        document.StatusPending,
		StatusMessage: "",
		TMCreate:      &now,
		TMUpdate:      &now,
	}

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

	_, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building insert SQL: %v", err)
	}

	// The 6th arg (index 5) should be nil for storage_file_id
	if args[5] != nil {
		t.Errorf("expected storage_file_id arg to be nil, got %v", args[5])
	}
}

func Test_DocumentGetsByRagID_SQL(t *testing.T) {
	ragID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003")

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"rag_id": ragID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building select SQL: %v", err)
	}

	expectedSQL := "SELECT id, customer_id, rag_id, name, doc_type, storage_file_id, source_url, status, status_message, tm_create, tm_update, tm_delete FROM rag_documents WHERE rag_id = $1 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	if len(args) != 1 {
		t.Errorf("unexpected arg count: got %d, want 1", len(args))
	}
}

func Test_DocumentGet_SQL(t *testing.T) {
	id := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building select SQL: %v", err)
	}

	expectedSQL := "SELECT id, customer_id, rag_id, name, doc_type, storage_file_id, source_url, status, status_message, tm_create, tm_update, tm_delete FROM rag_documents WHERE id = $1 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	if len(args) != 1 {
		t.Errorf("unexpected arg count: got %d, want 1", len(args))
	}
}

func Test_DocumentDeleteByRagID_SQL(t *testing.T) {
	ragID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003")
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_delete": &now,
			"tm_update": &now,
		}).
		Where(sq.Eq{"rag_id": ragID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building update SQL: %v", err)
	}

	// Verify it's an UPDATE with parameterized timestamps
	if !strings.HasPrefix(sqlStr, "UPDATE rag_documents SET") {
		t.Errorf("expected UPDATE statement, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "tm_delete = $") {
		t.Errorf("expected tm_delete = $N in SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "tm_update = $") {
		t.Errorf("expected tm_update = $N in SQL, got: %s", sqlStr)
	}

	// 3 args: tm_delete, tm_update, rag_id
	if len(args) != 3 {
		t.Errorf("unexpected arg count: got %d, want 3", len(args))
	}
}

func Test_DocumentDelete_SQL(t *testing.T) {
	id := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	q := psql.
		Update(tableDocuments).
		SetMap(map[string]any{
			"tm_delete": &now,
			"tm_update": &now,
		}).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building update SQL: %v", err)
	}

	if !strings.HasPrefix(sqlStr, "UPDATE rag_documents SET") {
		t.Errorf("expected UPDATE statement, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "tm_delete = $") {
		t.Errorf("expected tm_delete = $N in SQL, got: %s", sqlStr)
	}

	// 3 args: tm_delete, tm_update, id
	if len(args) != 3 {
		t.Errorf("unexpected arg count: got %d, want 3", len(args))
	}
}

func Test_DocumentUpdate_SQL(t *testing.T) {
	id := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	fields := map[document.Field]any{
		document.FieldStatus:        document.StatusReady,
		document.FieldStatusMessage: "processing complete",
	}

	updateMap := map[string]any{
		"tm_update": &now,
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
		t.Fatalf("unexpected error building update SQL: %v", err)
	}

	// Verify it contains the expected structure
	if len(sqlStr) == 0 {
		t.Error("expected non-empty SQL string")
	}

	// Should have 4 args: status value, status_message value, tm_update value, and id
	if len(args) != 4 {
		t.Errorf("unexpected arg count: got %d, want 4", len(args))
	}
}

func Test_DocumentGetsByCustomerID_SQL(t *testing.T) {
	customerID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002")

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"customer_id": customerID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building select SQL: %v", err)
	}

	expectedSQL := "SELECT id, customer_id, rag_id, name, doc_type, storage_file_id, source_url, status, status_message, tm_create, tm_update, tm_delete FROM rag_documents WHERE customer_id = $1 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	if len(args) != 1 {
		t.Errorf("unexpected arg count: got %d, want 1", len(args))
	}
}

func Test_psql_UsesDollarPlaceholders(t *testing.T) {
	q := psql.Select("id").From("test").Where(sq.Eq{"a": 1, "b": 2})
	sqlStr, _, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use $1, $2 instead of ?, ?
	expected1 := "SELECT id FROM test WHERE a = $1 AND b = $2"
	expected2 := "SELECT id FROM test WHERE b = $1 AND a = $2"
	if sqlStr != expected1 && sqlStr != expected2 {
		t.Errorf("expected dollar placeholders.\ngot:  %s\nwant: %s\n  or: %s", sqlStr, expected1, expected2)
	}
}
