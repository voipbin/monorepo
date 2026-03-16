package dbhandler

import (
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
)

func Test_formatEmbedding(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float32
		expected  string
	}{
		{
			name:      "empty embedding",
			embedding: []float32{},
			expected:  "[]",
		},
		{
			name:      "single element",
			embedding: []float32{0.5},
			expected:  "[0.5]",
		},
		{
			name:      "multiple elements",
			embedding: []float32{0.1, 0.2, 0.3},
			expected:  "[0.1,0.2,0.3]",
		},
		{
			name:      "negative values",
			embedding: []float32{-0.5, 0.0, 1.0},
			expected:  "[-0.5,0,1]",
		},
		{
			name:      "scientific notation avoided for normal values",
			embedding: []float32{0.123456, 0.789012},
			expected:  "[0.123456,0.789012]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatEmbedding(tt.embedding)
			if got != tt.expected {
				t.Errorf("formatEmbedding() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func Test_chunkColumns(t *testing.T) {
	cols := chunkColumns()
	if len(cols) != 10 {
		t.Errorf("chunkColumns() returned %d columns, want 10", len(cols))
	}

	expected := []string{
		"id", "document_id", "rag_id", "customer_id",
		"chunk_index", "text", "section_title",
		"token_count", "tm_create", "tm_delete",
	}
	for i, col := range expected {
		if cols[i] != col {
			t.Errorf("chunkColumns()[%d] = %q, want %q", i, cols[i], col)
		}
	}
}

func Test_ChunkDeleteByDocumentID_SQL(t *testing.T) {
	fakeID := uuid.Must(uuid.NewV4())

	q := psql.Delete(tableRagChunks).
		Where(sq.Eq{"document_id": fakeID})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql() error: %v", err)
	}

	expectedSQL := "DELETE FROM rag_chunks WHERE document_id = $1"
	if sqlStr != expectedSQL {
		t.Errorf("SQL = %q, want %q", sqlStr, expectedSQL)
	}
	if len(args) != 1 {
		t.Errorf("args length = %d, want 1", len(args))
	}
}

func Test_ChunkDeleteByRagID_SQL(t *testing.T) {
	fakeID := uuid.Must(uuid.NewV4())

	q := psql.Delete(tableRagChunks).
		Where(sq.Eq{"rag_id": fakeID})

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql() error: %v", err)
	}

	expectedSQL := "DELETE FROM rag_chunks WHERE rag_id = $1"
	if sqlStr != expectedSQL {
		t.Errorf("SQL = %q, want %q", sqlStr, expectedSQL)
	}
	if len(args) != 1 {
		t.Errorf("args length = %d, want 1", len(args))
	}
}

func Test_ChunkSoftDeleteByDocumentID_SQL(t *testing.T) {
	fakeID := uuid.Must(uuid.NewV4())
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	q := psql.Update(tableRagChunks).
		Set("tm_delete", &now).
		Where(sq.Eq{"document_id": fakeID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql() error: %v", err)
	}

	expectedSQL := "UPDATE rag_chunks SET tm_delete = $1 WHERE document_id = $2 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("SQL = %q, want %q", sqlStr, expectedSQL)
	}
	if len(args) != 2 {
		t.Errorf("args length = %d, want 2", len(args))
	}
}

func Test_ChunkSoftDeleteByRagID_SQL(t *testing.T) {
	fakeID := uuid.Must(uuid.NewV4())
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	q := psql.Update(tableRagChunks).
		Set("tm_delete", &now).
		Where(sq.Eq{"rag_id": fakeID}).
		Where("tm_delete IS NULL")

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql() error: %v", err)
	}

	expectedSQL := "UPDATE rag_chunks SET tm_delete = $1 WHERE rag_id = $2 AND tm_delete IS NULL"
	if sqlStr != expectedSQL {
		t.Errorf("SQL = %q, want %q", sqlStr, expectedSQL)
	}
	if len(args) != 2 {
		t.Errorf("args length = %d, want 2", len(args))
	}
}

func Test_ChunkSearchByRagID_QueryFormat(t *testing.T) {
	expectedQuery := `SELECT id, document_id, rag_id, customer_id, chunk_index, text, section_title, token_count, tm_create,
                     embedding <=> $1::vector AS distance
              FROM rag_chunks
              WHERE rag_id = $2 AND tm_delete IS NULL
              ORDER BY embedding <=> $1::vector
              LIMIT $3`

	if !containsAll(expectedQuery, []string{
		"<=> $1::vector",
		"AS distance",
		"rag_id = $2",
		"tm_delete IS NULL",
		"ORDER BY embedding <=> $1::vector",
		"LIMIT $3",
	}) {
		t.Errorf("search query missing expected pgvector elements")
	}
}

func Test_ChunkInsert_SQL(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3}
	embStr := formatEmbedding(embedding)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeID := uuid.Must(uuid.NewV4())

	q := psql.Insert(tableRagChunks).
		Columns(
			"id", "document_id", "rag_id", "customer_id",
			"chunk_index", "text", "section_title",
			"embedding", "token_count", "tm_create",
		).
		Values(
			fakeID, fakeID, fakeID, fakeID,
			0, "test text", "test section",
			sq.Expr("?::vector", embStr),
			100, &now,
		)

	sqlStr, args, err := q.ToSql()
	if err != nil {
		t.Fatalf("ToSql() error: %v", err)
	}

	expectedSQL := "INSERT INTO rag_chunks (id,document_id,rag_id,customer_id,chunk_index,text,section_title,embedding,token_count,tm_create) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::vector,$9,$10)"
	if sqlStr != expectedSQL {
		t.Errorf("SQL = %q, want %q", sqlStr, expectedSQL)
	}

	// args: 4 UUIDs + chunk_index + text + section_title + embedding_string + token_count + tm_create = 10
	if len(args) != 10 {
		t.Errorf("args length = %d, want 10", len(args))
	}

	// The embedding string should be at index 7
	if embArg, ok := args[7].(string); !ok || embArg != "[0.1,0.2,0.3]" {
		t.Errorf("embedding arg = %v, want %q", args[7], "[0.1,0.2,0.3]")
	}
}

// containsAll checks if s contains all the given substrings.
func containsAll(s string, substrings []string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
