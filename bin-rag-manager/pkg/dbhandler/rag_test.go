package dbhandler

import (
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/rag"
)

func Test_RagCreate_SQL(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	r := &rag.Rag{
		ID:          id,
		CustomerID:  customerID,
		Name:        "test-rag",
		Description: "test description",
		TMCreate:    &now,
		TMUpdate:    &now,
	}

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
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	expectedSQL := "INSERT INTO rag_rags (id,customer_id,name,description,tm_create,tm_update) VALUES ($1,$2,$3,$4,$5,$6)"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	// Should have 6 args: id, customer_id, name, description, tm_create, tm_update
	if len(args) != 6 {
		t.Errorf("unexpected number of args: got %d, want 6", len(args))
	}

	// Verify name and description are in args
	if args[2] != "test-rag" {
		t.Errorf("unexpected name arg: got %v, want test-rag", args[2])
	}
	if args[3] != "test description" {
		t.Errorf("unexpected description arg: got %v, want test description", args[3])
	}
}

func Test_RagGet_SQL(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	q := psql.
		Select(ragColumns()...).
		From(tableRagRags).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	expectedSQL := "SELECT id, customer_id, name, description, tm_create, tm_update, tm_delete FROM rag_rags WHERE id = $1 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	if len(args) != 1 {
		t.Errorf("unexpected number of args: got %d, want 1", len(args))
	}
}

func Test_RagGetsByCustomerID_SQL(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())

	q := psql.
		Select(ragColumns()...).
		From(tableRagRags).
		Where(sq.Eq{"customer_id": customerID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	expectedSQL := "SELECT id, customer_id, name, description, tm_create, tm_update, tm_delete FROM rag_rags WHERE customer_id = $1 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("unexpected SQL.\ngot:  %s\nwant: %s", sqlStr, expectedSQL)
	}

	if len(args) != 1 {
		t.Errorf("unexpected number of args: got %d, want 1", len(args))
	}
}

func Test_RagDelete_SQL(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	q := psql.
		Update(tableRagRags).
		SetMap(map[string]any{
			"tm_delete": &now,
			"tm_update": &now,
		}).
		Where(sq.Eq{"id": id}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	// Verify it's an UPDATE (soft delete), not a DELETE
	if !strings.HasPrefix(sqlStr, "UPDATE rag_rags SET") {
		t.Errorf("expected UPDATE statement, got: %s", sqlStr)
	}

	// Verify it sets tm_delete and tm_update as parameterized values
	if !strings.Contains(sqlStr, "tm_delete = $") {
		t.Errorf("expected tm_delete = $N in SQL, got: %s", sqlStr)
	}
	if !strings.Contains(sqlStr, "tm_update = $") {
		t.Errorf("expected tm_update = $N in SQL, got: %s", sqlStr)
	}

	// Verify WHERE clause includes id and tm_delete IS NULL
	if !strings.Contains(sqlStr, "tm_delete IS NULL") {
		t.Errorf("expected tm_delete IS NULL in WHERE clause, got: %s", sqlStr)
	}

	// Should have 3 args: tm_delete, tm_update, and id
	if len(args) != 3 {
		t.Errorf("unexpected number of args: got %d, want 3", len(args))
	}
}

func Test_RagUpdate_SQL(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	fields := map[rag.Field]any{
		rag.FieldName: "updated-name",
	}

	updateMap := map[string]any{
		"tm_update": &now,
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
		t.Fatalf("unexpected error building SQL: %v", err)
	}

	// Verify it's an UPDATE statement
	if !strings.HasPrefix(sqlStr, "UPDATE rag_rags SET") {
		t.Errorf("expected UPDATE statement, got: %s", sqlStr)
	}

	// Verify the field value is set
	if !strings.Contains(sqlStr, "name =") {
		t.Errorf("expected name field in SET clause, got: %s", sqlStr)
	}

	// Verify WHERE clause includes tm_delete IS NULL
	if !strings.Contains(sqlStr, "tm_delete IS NULL") {
		t.Errorf("expected tm_delete IS NULL in WHERE clause, got: %s", sqlStr)
	}

	// Args should include tm_update value, name value, and id
	if len(args) != 3 {
		t.Errorf("unexpected number of args: got %d, want 3", len(args))
	}
}

func Test_RagColumns(t *testing.T) {
	cols := ragColumns()

	expected := []string{"id", "customer_id", "name", "description", "tm_create", "tm_update", "tm_delete"}

	if len(cols) != len(expected) {
		t.Fatalf("unexpected number of columns: got %d, want %d", len(cols), len(expected))
	}

	for i, col := range cols {
		if col != expected[i] {
			t.Errorf("column %d: got %s, want %s", i, col, expected[i])
		}
	}
}
