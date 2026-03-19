# RAG Document Ingestion Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement an async ingestion pipeline in rag-manager that fetches files from GCS or URLs, chunks them, generates embeddings, and stores indexed chunks — with a simplified API where RAG creation includes sources and documents are internal.

**Architecture:** `RagCreate` now accepts file IDs and URLs, internally creates documents, and launches async ingestion goroutines. Each goroutine atomically claims a document, downloads the file to a temp file, chunks by format, embeds via Google Gemini, and stores chunks in PostgreSQL with pgvector. RAG responses include computed status and per-source progress. Startup sweep and periodic ticker recover stuck documents.

**Tech Stack:** Go, PostgreSQL/pgvector, Google Cloud Storage, Google Gemini text-embedding-004, RabbitMQ RPC, Squirrel query builder

**Design doc:** `docs/plans/2026-03-19-rag-document-ingestion-pipeline-design.md`

---

### Task 1: Database Migration — Add Ingestion Fields

**Files:**
- Create: `bin-rag-manager/migrations/000002_add_document_ingestion_fields.up.sql`
- Create: `bin-rag-manager/migrations/000002_add_document_ingestion_fields.down.sql`

**Step 1: Create the up migration**

```sql
-- 000002_add_document_ingestion_fields.up.sql
ALTER TABLE rag_documents ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE rag_documents ADD COLUMN IF NOT EXISTS tm_processing TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_rag_documents_status ON rag_documents(status) WHERE tm_delete IS NULL;
```

**Step 2: Create the down migration**

```sql
-- 000002_add_document_ingestion_fields.down.sql
DROP INDEX IF EXISTS idx_rag_documents_status;
ALTER TABLE rag_documents DROP COLUMN IF EXISTS tm_processing;
ALTER TABLE rag_documents DROP COLUMN IF EXISTS retry_count;
```

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline
git add bin-rag-manager/migrations/000002_add_document_ingestion_fields.up.sql \
        bin-rag-manager/migrations/000002_add_document_ingestion_fields.down.sql
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add migration for retry_count and tm_processing columns"
```

---

### Task 2: Document Model and Field Updates

**Files:**
- Modify: `bin-rag-manager/models/document/main.go`
- Modify: `bin-rag-manager/models/document/field.go`

**Step 1: Add new fields to Document struct**

In `bin-rag-manager/models/document/main.go`, add after `TMDelete`:

```go
RetryCount    int        `json:"retry_count,omitempty" db:"retry_count"`
TMProcessing  *time.Time `json:"tm_processing,omitempty" db:"tm_processing"`
```

**Step 2: Add new field constants**

In `bin-rag-manager/models/document/field.go`, add to the const block:

```go
FieldRetryCount    Field = "retry_count"
FieldTMProcessing  Field = "tm_processing"
```

**Step 3: Commit**

```bash
git add bin-rag-manager/models/document/main.go bin-rag-manager/models/document/field.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add RetryCount and TMProcessing to Document model"
```

---

### Task 3: RAG Model — Add Transient Fields and Update Webhook

**Files:**
- Modify: `bin-rag-manager/models/rag/main.go`
- Modify: `bin-rag-manager/models/rag/webhook.go`

The RAG response includes a computed `Status` and a `Sources` list. These are **transient fields** on the Rag struct (no `db:` tag) — populated by raghandler, ignored by DB operations, serialized over RPC via JSON.

**Note on cross-package import (C4):** `models/rag` imports `models/document` for the `Status` type. This is a same-module import (`bin-rag-manager`), not a cross-module dependency — perfectly fine. Creating a separate `rag.Status` type would duplicate constants and add conversion noise.

**Step 1: Add Source type and transient fields to `main.go`**

Add after the existing `Rag` struct:

```go
import (
	"time"

	"github.com/gofrs/uuid"

	rmdocument "monorepo/bin-rag-manager/models/document"
)

// Source represents a single source (document) in the RAG response.
type Source struct {
	StorageFileID *uuid.UUID       `json:"storage_file_id,omitempty"`
	SourceURL     string           `json:"source_url,omitempty"`
	Status        rmdocument.Status `json:"status,omitempty"`
	StatusMessage string           `json:"status_message,omitempty"`
}
```

Add transient fields to the `Rag` struct (after `TMDelete`, no `db:` tag):

```go
// Transient — populated by handler, ignored by DB (no db tag)
Status  rmdocument.Status `json:"status,omitempty"`
Sources []Source          `json:"sources,omitempty"`
```

**Step 2: Update WebhookMessage in `webhook.go`**

Replace the existing `webhook.go` content. `ConvertWebhookMessage` now copies Status and Sources through naturally.

```go
package rag

import (
	"time"

	"github.com/gofrs/uuid"

	rmdocument "monorepo/bin-rag-manager/models/document"
)

// WebhookMessage is the external-facing representation of a RAG.
type WebhookMessage struct {
	ID          uuid.UUID         `json:"id,omitempty"`
	CustomerID  uuid.UUID         `json:"customer_id,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      rmdocument.Status `json:"status,omitempty"`
	Sources     []Source          `json:"sources,omitempty"`
	TMCreate    *time.Time        `json:"tm_create,omitempty"`
	TMUpdate    *time.Time        `json:"tm_update,omitempty"`
}

// ConvertWebhookMessage converts internal Rag to external representation.
// Status and Sources are copied from transient fields (populated by raghandler).
func (r *Rag) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:          r.ID,
		CustomerID:  r.CustomerID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		Sources:     r.Sources,
		TMCreate:    r.TMCreate,
		TMUpdate:    r.TMUpdate,
	}
}
```

**Step 2: Run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go test ./models/rag/... -v
```

**Step 3: Commit**

```bash
git add bin-rag-manager/models/rag/webhook.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add Status and Sources to RAG WebhookMessage"
```

---

### Task 4: DBHandler — Update Document Scan and Add New Methods

**Files:**
- Modify: `bin-rag-manager/pkg/dbhandler/main.go`
- Modify: `bin-rag-manager/pkg/dbhandler/document.go`

**Step 1: Update `documentColumns()` to include new columns**

In `bin-rag-manager/pkg/dbhandler/document.go`, add `"retry_count"` and `"tm_processing"` to the end of the `documentColumns()` return slice.

**Step 2: Update `scanDocument` to scan new fields**

In the `scanDocument` function, add to the end of the `Scan` call:

```go
&d.RetryCount,
&d.TMProcessing,
```

Do the same in `scanDocumentRows`.

**Step 3: Add new methods to DBHandler interface**

In `bin-rag-manager/pkg/dbhandler/main.go`, add to the `DBHandler` interface after the existing Document methods:

```go
DocumentClaimForProcessing(ctx context.Context, id uuid.UUID) (*document.Document, error)
DocumentUpdateHeartbeat(ctx context.Context, id uuid.UUID) error
DocumentGetStale(ctx context.Context, threshold time.Duration) ([]*document.Document, error)
DocumentGetPending(ctx context.Context) ([]*document.Document, error)
DocumentResetStaleToPending(ctx context.Context, threshold time.Duration) error
DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error)
DocumentGetsByRagIDs(ctx context.Context, ragIDs []uuid.UUID) (map[uuid.UUID][]*document.Document, error)
```

Add `"time"` to the imports.

**Step 4: Implement `DocumentClaimForProcessing`**

Atomically claim a pending document for processing. Only succeeds if the document is still in `pending` status.

**Note:** This method uses `strings.Join` — add `"strings"` to the imports in `document.go` if not already present.

```go
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
```

**Step 5: Implement `DocumentUpdateHeartbeat`**

```go
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
```

**Step 6: Implement `DocumentGetStale`**

Find documents stuck in `processing` with heartbeat older than threshold.

```go
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
```

**Step 7: Implement `DocumentGetPending`**

Find pending documents with retry_count < 3.

```go
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
```

**Step 8: Implement `DocumentResetStaleToPending`**

Reset stale processing documents back to pending.

```go
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
```

**Step 9: Implement `DocumentGetsByRagID`**

Get all active documents for a RAG (for building sources in RAG response).

```go
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
```

**Step 10: Implement `DocumentGetsByRagIDs` (batch)**

Batch fetch documents for multiple RAGs (used by RagList to avoid N+1).

```go
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
```

**Step 11: Run tests and generate mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go generate ./...
go test ./...
golangci-lint run -v --timeout 5m
```

**Step 12: Commit**

```bash
git add bin-rag-manager/pkg/dbhandler/main.go bin-rag-manager/pkg/dbhandler/document.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Update document scan for retry_count and tm_processing columns
- bin-rag-manager: Add DocumentClaimForProcessing, heartbeat, stale, pending, GetsByRagID, GetsByRagIDs DB methods"
```

---

### Task 5: New Chunkers — Plain Text, CSV, JSON

These use only stdlib — no new dependencies.

**Files:**
- Create: `bin-rag-manager/pkg/chunker/text.go`
- Create: `bin-rag-manager/pkg/chunker/text_test.go`
- Create: `bin-rag-manager/pkg/chunker/csv.go`
- Create: `bin-rag-manager/pkg/chunker/csv_test.go`
- Create: `bin-rag-manager/pkg/chunker/json_chunker.go`
- Create: `bin-rag-manager/pkg/chunker/json_chunker_test.go`

**Context:** The `splitByTokenLimit` and `generateChunkID` helper functions already exist in `pkg/chunker/rst.go` (lines 136-168) and are available to all files in the `chunker` package. Use them directly — do NOT duplicate.

**Step 1: Write text chunker test**

```go
// text_test.go
package chunker

import (
	"os"
	"strings"
	"testing"
)

func TestTextChunker_Chunk(t *testing.T) {
	content := "This is a plain text document.\n\nIt has multiple paragraphs.\n\nEach paragraph should be chunked properly."

	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewTextChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}

	var allText strings.Builder
	for _, chunk := range chunks {
		allText.WriteString(chunk.Text)
	}
	if !strings.Contains(allText.String(), "plain text document") {
		t.Error("expected content to be preserved")
	}
}

func TestTextChunker_NonExistentFile(t *testing.T) {
	c := NewTextChunker()
	_, err := c.Chunk("/nonexistent/file.txt", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestTextChunker_LargeFile(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("This is paragraph number ")
		sb.WriteString(strings.Repeat("word ", 50))
		sb.WriteString("\n\n")
	}

	tmpFile, err := os.CreateTemp("", "test_large_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewTextChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 100)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks for large file, got %d", len(chunks))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go test ./pkg/chunker/ -run TestTextChunker -v
```
Expected: FAIL — `NewTextChunker` undefined.

**Step 3: Implement text chunker**

```go
// text.go
package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type textChunker struct{}

// NewTextChunker creates a chunker for plain text files.
func NewTextChunker() Chunker {
	return &textChunker{}
}

func (c *textChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	textStr := string(content)
	if strings.TrimSpace(textStr) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(textStr, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, text := range parts {
		title := fmt.Sprintf("Part %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         text,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
```

**Step 4: Run text tests**

```bash
go test ./pkg/chunker/ -run TestTextChunker -v
```
Expected: PASS

**Step 5: Write CSV chunker test**

```go
// csv_test.go
package chunker

import (
	"os"
	"testing"
)

func TestCSVChunker_Chunk(t *testing.T) {
	content := "name,email,role\nAlice,alice@example.com,admin\nBob,bob@example.com,user\nCharlie,charlie@example.com,user\n"

	tmpFile, err := os.CreateTemp("", "test_*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewCSVChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}

	for _, chunk := range chunks {
		if chunk.Text == "" {
			t.Error("expected non-empty chunk text")
		}
	}
}

func TestCSVChunker_NonExistentFile(t *testing.T) {
	c := NewCSVChunker()
	_, err := c.Chunk("/nonexistent/file.csv", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
```

**Step 6: Implement CSV chunker**

```go
// csv.go
package chunker

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type csvChunker struct{}

// NewCSVChunker creates a chunker for CSV files.
func NewCSVChunker() Chunker {
	return &csvChunker{}
}

func (c *csvChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not parse CSV %s: %w", filePath, err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	header := records[0]
	headerLine := strings.Join(header, ",")
	maxChars := maxTokens * 4

	chunks := []Chunk{}
	var current strings.Builder
	current.WriteString(headerLine)
	current.WriteString("\n")
	chunkIdx := 0

	for _, row := range records[1:] {
		rowLine := strings.Join(row, ",")
		if current.Len()+len(rowLine)+1 > maxChars && current.Len() > len(headerLine)+1 {
			title := fmt.Sprintf("Rows (part %d)", chunkIdx+1)
			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         current.String(),
				SourceFile:   filePath,
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
			chunkIdx++
			current.Reset()
			current.WriteString(headerLine)
			current.WriteString("\n")
		}
		current.WriteString(rowLine)
		current.WriteString("\n")
	}

	if current.Len() > len(headerLine)+1 {
		title := fmt.Sprintf("Rows (part %d)", chunkIdx+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         current.String(),
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
```

**Step 7: Run CSV tests**

```bash
go test ./pkg/chunker/ -run TestCSVChunker -v
```
Expected: PASS

**Step 8: Write JSON chunker test**

```go
// json_chunker_test.go
package chunker

import (
	"os"
	"testing"
)

func TestJSONChunker_Object(t *testing.T) {
	content := `{"users": [{"name": "Alice"}, {"name": "Bob"}], "settings": {"theme": "dark"}}`

	tmpFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewJSONChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}
}

func TestJSONChunker_Array(t *testing.T) {
	content := `[{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]`

	tmpFile, err := os.CreateTemp("", "test_*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewJSONChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}
}

func TestJSONChunker_NonExistentFile(t *testing.T) {
	c := NewJSONChunker()
	_, err := c.Chunk("/nonexistent/file.json", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
```

**Step 9: Implement JSON chunker**

```go
// json_chunker.go
package chunker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type jsonChunker struct{}

// NewJSONChunker creates a chunker for JSON files.
func NewJSONChunker() Chunker {
	return &jsonChunker{}
}

func (c *jsonChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", filePath, err)
	}

	relPath := filepath.Base(filePath)

	// Try to parse as JSON object
	var obj map[string]any
	if err := json.Unmarshal(content, &obj); err == nil {
		return c.chunkObject(relPath, filePath, obj, maxTokens)
	}

	// Try to parse as JSON array
	var arr []any
	if err := json.Unmarshal(content, &arr); err == nil {
		return c.chunkArray(relPath, filePath, arr, maxTokens)
	}

	// Fallback: treat as plain text
	parts := splitByTokenLimit(string(content), maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, text := range parts {
		title := fmt.Sprintf("Part %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         text,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}

func (c *jsonChunker) chunkObject(relPath, filePath string, obj map[string]any, maxTokens int) ([]Chunk, error) {
	chunks := []Chunk{}
	maxChars := maxTokens * 4

	for key, val := range obj {
		b, err := json.MarshalIndent(map[string]any{key: val}, "", "  ")
		if err != nil {
			continue
		}

		text := string(b)
		if len(text) <= maxChars {
			title := fmt.Sprintf("Key: %s", key)
			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         text,
				SourceFile:   filePath,
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
		} else {
			parts := splitByTokenLimit(text, maxTokens)
			for i, part := range parts {
				title := fmt.Sprintf("Key: %s (part %d)", key, i+1)
				id := generateChunkID(relPath, title)
				chunks = append(chunks, Chunk{
					ID:           id,
					Text:         part,
					SourceFile:   filePath,
					DocType:      DocTypeDevDoc,
					SectionTitle: title,
				})
			}
		}
	}

	return chunks, nil
}

func (c *jsonChunker) chunkArray(relPath, filePath string, arr []any, maxTokens int) ([]Chunk, error) {
	maxChars := maxTokens * 4
	chunks := []Chunk{}
	var current []any
	currentSize := 0
	chunkIdx := 0

	for _, item := range arr {
		b, err := json.Marshal(item)
		if err != nil {
			continue
		}

		if currentSize+len(b) > maxChars && len(current) > 0 {
			text, _ := json.MarshalIndent(current, "", "  ")
			title := fmt.Sprintf("Items (part %d)", chunkIdx+1)
			id := generateChunkID(relPath, title)
			chunks = append(chunks, Chunk{
				ID:           id,
				Text:         string(text),
				SourceFile:   filePath,
				DocType:      DocTypeDevDoc,
				SectionTitle: title,
			})
			chunkIdx++
			current = nil
			currentSize = 0
		}
		current = append(current, item)
		currentSize += len(b)
	}

	if len(current) > 0 {
		text, _ := json.MarshalIndent(current, "", "  ")
		title := fmt.Sprintf("Items (part %d)", chunkIdx+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         string(text),
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
```

**Step 10: Run all chunker tests**

```bash
go test ./pkg/chunker/ -v
```
Expected: ALL PASS

**Step 11: Commit**

```bash
git add bin-rag-manager/pkg/chunker/text.go bin-rag-manager/pkg/chunker/text_test.go \
        bin-rag-manager/pkg/chunker/csv.go bin-rag-manager/pkg/chunker/csv_test.go \
        bin-rag-manager/pkg/chunker/json_chunker.go bin-rag-manager/pkg/chunker/json_chunker_test.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add text, CSV, and JSON chunkers with tests"
```

---

### Task 6: New Chunkers — HTML, PDF, DOCX

These require external dependencies.

**Files:**
- Create: `bin-rag-manager/pkg/chunker/html.go`
- Create: `bin-rag-manager/pkg/chunker/html_test.go`
- Create: `bin-rag-manager/pkg/chunker/pdf.go`
- Create: `bin-rag-manager/pkg/chunker/pdf_test.go`
- Create: `bin-rag-manager/pkg/chunker/docx.go`
- Create: `bin-rag-manager/pkg/chunker/docx_test.go`

**Step 1: Add external dependencies**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go get golang.org/x/net/html
go get github.com/ledongthuc/pdf
go get github.com/fumiama/go-docx
```

**Step 2: Write HTML chunker test**

```go
// html_test.go
package chunker

import (
	"os"
	"strings"
	"testing"
)

func TestHTMLChunker_Chunk(t *testing.T) {
	content := `<html><body><h1>Title</h1><p>Paragraph one.</p><h2>Section</h2><p>Paragraph two.</p></body></html>`

	tmpFile, err := os.CreateTemp("", "test_*.html")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpFile.Close()

	c := NewHTMLChunker()
	chunks, err := c.Chunk(tmpFile.Name(), 800)
	if err != nil {
		t.Fatalf("chunk error: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("expected at least 1 chunk")
	}

	for _, chunk := range chunks {
		if strings.Contains(chunk.Text, "<html>") {
			t.Error("expected HTML tags to be stripped")
		}
	}
}

func TestHTMLChunker_NonExistentFile(t *testing.T) {
	c := NewHTMLChunker()
	_, err := c.Chunk("/nonexistent/file.html", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
```

**Step 3: Implement HTML chunker**

```go
// html.go
package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

type htmlChunker struct{}

// NewHTMLChunker creates a chunker for HTML files.
func NewHTMLChunker() Chunker {
	return &htmlChunker{}
}

func (c *htmlChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	doc, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse HTML %s: %w", filePath, err)
	}

	text := extractHTMLText(doc)
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(text, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, part := range parts {
		title := fmt.Sprintf("Part %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         part,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}

func extractHTMLText(n *html.Node) string {
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return ""
	}

	if n.Type == html.TextNode {
		return n.Data
	}

	var sb strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		sb.WriteString(extractHTMLText(child))
		if child.Type == html.ElementNode {
			switch child.Data {
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "br", "li", "tr":
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}
```

**Step 4: Run HTML tests**

```bash
go test ./pkg/chunker/ -run TestHTMLChunker -v
```
Expected: PASS

**Step 5: Write PDF chunker test**

```go
// pdf_test.go
package chunker

import (
	"testing"
)

func TestPDFChunker_NonExistentFile(t *testing.T) {
	c := NewPDFChunker()
	_, err := c.Chunk("/nonexistent/file.pdf", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
```

Note: Testing PDF parsing with real content requires a valid PDF binary file. The non-existent file test validates error handling. Integration testing with real PDFs should be done manually.

**Step 6: Implement PDF chunker**

```go
// pdf.go
package chunker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

type pdfChunker struct{}

// NewPDFChunker creates a chunker for PDF files.
func NewPDFChunker() Chunker {
	return &pdfChunker{}
}

func (c *pdfChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open PDF %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	var sb strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}

	textStr := sb.String()
	if strings.TrimSpace(textStr) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(textStr, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, text := range parts {
		title := fmt.Sprintf("Page group %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         text,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
```

**Step 7: Write DOCX chunker test**

```go
// docx_test.go
package chunker

import (
	"testing"
)

func TestDOCXChunker_NonExistentFile(t *testing.T) {
	c := NewDOCXChunker()
	_, err := c.Chunk("/nonexistent/file.docx", 800)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
```

**Step 8: Implement DOCX chunker**

```go
// docx.go
package chunker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fumiama/go-docx"
)

type docxChunker struct{}

// NewDOCXChunker creates a chunker for DOCX files.
func NewDOCXChunker() Chunker {
	return &docxChunker{}
}

func (c *docxChunker) Chunk(filePath string, maxTokens int) ([]Chunk, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not stat file %s: %w", filePath, err)
	}

	doc, err := docx.Parse(f, fi.Size())
	if err != nil {
		return nil, fmt.Errorf("could not parse DOCX %s: %w", filePath, err)
	}

	var sb strings.Builder
	for _, item := range doc.Document.Body.Items {
		p, ok := item.(*docx.Paragraph)
		if !ok {
			continue
		}
		for _, child := range p.Children {
			r, ok := child.(*docx.Run)
			if !ok {
				continue
			}
			if r.Text != nil {
				sb.WriteString(r.Text.Text)
			}
		}
		sb.WriteString("\n")
	}

	textStr := sb.String()
	if strings.TrimSpace(textStr) == "" {
		return nil, nil
	}

	relPath := filepath.Base(filePath)
	parts := splitByTokenLimit(textStr, maxTokens)
	chunks := make([]Chunk, 0, len(parts))
	for i, text := range parts {
		title := fmt.Sprintf("Part %d", i+1)
		id := generateChunkID(relPath, title)
		chunks = append(chunks, Chunk{
			ID:           id,
			Text:         text,
			SourceFile:   filePath,
			DocType:      DocTypeDevDoc,
			SectionTitle: title,
		})
	}

	return chunks, nil
}
```

**Step 9: Run all chunker tests + tidy**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go mod tidy
go test ./pkg/chunker/ -v
```
Expected: ALL PASS

**Step 10: Commit**

```bash
git add bin-rag-manager/pkg/chunker/html.go bin-rag-manager/pkg/chunker/html_test.go \
        bin-rag-manager/pkg/chunker/pdf.go bin-rag-manager/pkg/chunker/pdf_test.go \
        bin-rag-manager/pkg/chunker/docx.go bin-rag-manager/pkg/chunker/docx_test.go \
        bin-rag-manager/go.mod bin-rag-manager/go.sum
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add HTML, PDF, and DOCX chunkers with tests"
```

---

### Task 7: Chunker Selector — Format Dispatcher

**Files:**
- Create: `bin-rag-manager/pkg/chunker/selector.go`
- Create: `bin-rag-manager/pkg/chunker/selector_test.go`

**Step 1: Write selector test**

```go
// selector_test.go
package chunker

import (
	"testing"
)

func TestGetChunkerByExtension(t *testing.T) {
	tests := []struct {
		ext      string
		wantNil  bool
	}{
		{".rst", false},
		{".md", false},
		{".yaml", false},
		{".yml", false},
		{".txt", false},
		{".pdf", false},
		{".html", false},
		{".htm", false},
		{".csv", false},
		{".docx", false},
		{".json", false},
		{".unknown", false}, // fallback to text
		{"", false},         // fallback to text
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			c := GetChunkerByExtension(tt.ext)
			if (c == nil) != tt.wantNil {
				t.Errorf("GetChunkerByExtension(%q) nil=%v, want nil=%v", tt.ext, c == nil, tt.wantNil)
			}
		})
	}
}

func TestDetectExtensionFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"document.pdf", ".pdf"},
		{"readme.md", ".md"},
		{"data.CSV", ".csv"},
		{"noext", ""},
		{"archive.tar.gz", ".gz"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := DetectExtensionFromFilename(tt.filename)
			if got != tt.want {
				t.Errorf("DetectExtensionFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDetectExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        string
	}{
		{"application/pdf", ".pdf"},
		{"text/html", ".html"},
		{"text/html; charset=utf-8", ".html"},
		{"text/csv", ".csv"},
		{"application/json", ".json"},
		{"application/octet-stream", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := DetectExtensionFromContentType(tt.contentType)
			if got != tt.want {
				t.Errorf("DetectExtensionFromContentType(%q) = %q, want %q", tt.contentType, got, tt.want)
			}
		})
	}
}
```

**Step 2: Implement selector**

```go
// selector.go
package chunker

import (
	"path/filepath"
	"strings"
)

// contentTypeToExt maps HTTP Content-Type to file extension.
var contentTypeToExt = map[string]string{
	"text/plain":       ".txt",
	"text/html":        ".html",
	"text/csv":         ".csv",
	"text/markdown":    ".md",
	"application/pdf":  ".pdf",
	"application/json": ".json",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"text/x-rst":            ".rst",
	"text/restructuredtext":  ".rst",
	"application/x-yaml":    ".yaml",
	"text/yaml":             ".yaml",
}

// GetChunkerByExtension returns the appropriate chunker for the given file extension.
// Falls back to plain text chunker if the extension is not recognized.
func GetChunkerByExtension(ext string) Chunker {
	switch strings.ToLower(ext) {
	case ".rst":
		return NewRSTChunker()
	case ".md":
		return NewMarkdownChunker(DocTypeDevDoc)
	case ".yaml", ".yml":
		return NewOpenAPIChunker()
	case ".txt":
		return NewTextChunker()
	case ".pdf":
		return NewPDFChunker()
	case ".html", ".htm":
		return NewHTMLChunker()
	case ".csv":
		return NewCSVChunker()
	case ".docx":
		return NewDOCXChunker()
	case ".json":
		return NewJSONChunker()
	default:
		return NewTextChunker()
	}
}

// DetectExtensionFromFilename extracts the file extension from a filename.
// Returns lowercase extension with leading dot, or "" if none found.
func DetectExtensionFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

// DetectExtensionFromContentType maps an HTTP Content-Type header to a file extension.
// Returns "" if the content type is not recognized.
func DetectExtensionFromContentType(contentType string) string {
	// Strip charset and other parameters
	ct := contentType
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = strings.ToLower(ct)

	if ext, ok := contentTypeToExt[ct]; ok {
		return ext
	}
	return ""
}
```

**Step 3: Run selector tests**

```bash
go test ./pkg/chunker/ -run "TestGetChunker|TestDetectExtension" -v
```
Expected: ALL PASS

**Step 4: Commit**

```bash
git add bin-rag-manager/pkg/chunker/selector.go bin-rag-manager/pkg/chunker/selector_test.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add chunker selector with format detection by extension and content-type"
```

---

### Task 8: BucketReader — GCS Interface

**Files:**
- Create: `bin-rag-manager/pkg/bucketreader/main.go`

**Step 1: Add GCS dependency**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go get cloud.google.com/go/storage
```

**Step 2: Implement BucketReader interface**

```go
// main.go
package bucketreader

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
)

//go:generate mockgen -package bucketreader -source main.go -destination mock_main.go

// BucketReader reads files from GCS buckets.
type BucketReader interface {
	DownloadToTempFile(ctx context.Context, bucketName, filepath string) (tmpPath string, err error)
}

type bucketReader struct {
	client *storage.Client
}

// NewBucketReader creates a new BucketReader with the given GCS client.
func NewBucketReader(client *storage.Client) BucketReader {
	return &bucketReader{client: client}
}

func (b *bucketReader) DownloadToTempFile(ctx context.Context, bucketName, filepath string) (string, error) {
	reader, err := b.client.Bucket(bucketName).Object(filepath).NewReader(ctx)
	if err != nil {
		return "", fmt.Errorf("could not open GCS object %s/%s: %w", bucketName, filepath, err)
	}
	defer func() { _ = reader.Close() }()

	tmpFile, err := os.CreateTemp("", "rag_gcs_*")
	if err != nil {
		return "", fmt.Errorf("could not create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, reader); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("could not download GCS object: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("could not close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}
```

**Step 3: Generate mock and tidy**

```bash
go mod tidy
go generate ./pkg/bucketreader/...
```

**Step 4: Commit**

```bash
git add bin-rag-manager/pkg/bucketreader/ bin-rag-manager/go.mod bin-rag-manager/go.sum
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add BucketReader interface for GCS file download"
```

---

### Task 9: Config — Add GCP Bucket Name

**Files:**
- Modify: `bin-rag-manager/internal/config/config.go`
- Modify: `bin-rag-manager/cmd/rag-manager/main.go`

**Step 1: Add field and flag to config**

Add `GCPBucketNameMedia string` to the Config struct.

In `Bootstrap()`, add:
- Flag: `f.String("gcp_bucket_name_media", "", "GCS bucket name for media files")`
- Binding: `"gcp_bucket_name_media": "GCP_BUCKET_NAME_MEDIA"` to the bindings map

**IMPORTANT (H8 fix):** Config has **two init paths** that both populate `globalConfig`. Update **both**:

In `LoadGlobalConfig()`, add: `GCPBucketNameMedia: viper.GetString("gcp_bucket_name_media"),`

In `InitConfig()`, add the flag binding AND add to the config struct: `GCPBucketNameMedia: viper.GetString("gcp_bucket_name_media"),`

**Step 2: Register the flag in `cmd/rag-manager/main.go` `init()`**

**CRITICAL:** `InitConfig` binds pflags from `cmd.Flags()`. If the flag is never registered on `rootCmd`, `viper.GetString("gcp_bucket_name_media")` will always return `""`. Add this line to `init()` alongside the existing flags:

```go
rootCmd.Flags().String("gcp_bucket_name_media", "", "GCS bucket name for media files")
```

**Step 3: Commit**

```bash
git add bin-rag-manager/internal/config/config.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add gcp_bucket_name_media config flag"
```

---

### Task 10: RagHandler — Update Struct, Constructor, Interface

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/main.go`

**Step 1: Update the ragHandler struct**

Add `reqHandler`, `bucketReader`, and `bucketName` fields:

```go
import (
	// ... existing imports ...
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-rag-manager/pkg/bucketreader"
)

type ragHandler struct {
	embedder     embedder.Embedder
	dbHandler    dbhandler.DBHandler
	reqHandler   requesthandler.RequestHandler
	bucketReader bucketreader.BucketReader
	bucketName   string
}
```

**Step 2: Update the constructor**

```go
func NewRagHandler(
	emb embedder.Embedder,
	dbH dbhandler.DBHandler,
	reqH requesthandler.RequestHandler,
	br bucketreader.BucketReader,
	bucketName string,
) RagHandler {
	return &ragHandler{
		embedder:     emb,
		dbHandler:    dbH,
		reqHandler:   reqH,
		bucketReader: br,
		bucketName:   bucketName,
	}
}
```

**Step 3: Update the RagHandler interface**

Replace the existing interface with:

```go
type RagHandler interface {
	RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error)
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error)
	RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error)
	RagDelete(ctx context.Context, id uuid.UUID) error
	RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error)

	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error)

	QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error)

	DocumentIngestPendingAll(ctx context.Context)
	RunIngestionTicker(ctx context.Context, interval time.Duration)
}
```

Note: `DocumentCreate` and `DocumentDelete` are removed from the public interface — documents are now managed internally via `RagCreate` and `RagAddSources`. `RagGet` and `RagList` return enriched `*rag.Rag` with transient Status/Sources fields populated. No `RagGetWithSources` needed — `RagGet` does the enrichment directly.

**Step 4: Run generate to update mocks**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go generate ./...
```

**IMPORTANT:** This will NOT compile on its own — the interface changes reference methods that don't exist yet. **Do NOT commit Task 10 separately.** Tasks 10, 11, 12, 13, and 14 MUST be implemented together and committed as a single unit after the full verification workflow passes. The separate task numbers are for organizational clarity only. (Task 13 is included because the existing `listenhandler/v1_documents.go` calls `h.ragHandler.DocumentCreate()` and `h.ragHandler.DocumentDelete()` which are removed from the interface in Task 10.)

**Step 5: Update `raghandler/main_test.go`**

The existing test calls `NewRagHandler(nil, nil)` with 2 arguments. After the constructor change to 5 arguments, update:

```go
// In TestNewRagHandler:
h := NewRagHandler(nil, nil, nil, nil, "")
```

**Step 6: Stage changes (do NOT commit yet — wait for Tasks 11-14)**

```bash
# Stage but don't commit — must combine with Tasks 11-14
git add bin-rag-manager/pkg/raghandler/main.go bin-rag-manager/pkg/raghandler/main_test.go
# Commit message for reference (use when committing Tasks 10-14 together):
# git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Update RagHandler struct, constructor, and interface for ingestion pipeline"
```

---

### Task 11: RagHandler — Ingestion Pipeline, File Acquisition, Sources

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/rag.go`
- Modify: `bin-rag-manager/pkg/raghandler/document.go`

This is the core task. It implements:
- Updated `RagCreate` that accepts file IDs/URLs and creates documents internally
- Updated `RagGet` that enriches `*rag.Rag` with transient Status/Sources
- Updated `RagList` that enriches via batch document fetch
- `RagAddSources` for adding sources to existing RAGs
- `documentIngest` — the async ingestion pipeline
- File acquisition (GCS and URL download)
- Status computation from documents

**Step 1: Update `rag.go` — RagCreate with sources**

Replace the existing `RagCreate` implementation. **Match existing pattern:** each function declares `log := logrus.WithFields(...)` using `"github.com/sirupsen/logrus"` (unaliased). Do NOT use `log "github.com/sirupsen/logrus"` — the `log` variable is function-scoped.

```go
func (h *ragHandler) RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": customerID,
		"name":        name,
	})

	id, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate UUID. err: %v", err)
		return nil, fmt.Errorf("could not generate rag id: %w", err)
	}

	r := &rag.Rag{
		ID:          id,
		CustomerID:  customerID,
		Name:        name,
		Description: description,
	}

	if errCreate := h.dbHandler.RagCreate(ctx, r); errCreate != nil {
		log.Errorf("Could not create rag. err: %v", errCreate)
		return nil, fmt.Errorf("could not create rag: %w", errCreate)
	}
	log.WithField("rag", r).Debugf("Created rag. rag_id: %s", r.ID)

	// Create documents for each source and trigger ingestion
	h.createDocumentsForSources(ctx, customerID, id, storageFileIDs, sourceURLs)

	// Return enriched Rag with Status/Sources populated
	return h.RagGet(ctx, id)
}
```

**Step 2: Update `RagGet` to enrich with Status/Sources**

Update the existing `RagGet` to fetch documents and populate transient fields:

```go
func (h *ragHandler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagGet",
		"id":   id,
	})

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return nil, fmt.Errorf("could not get rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Retrieved rag. rag_id: %s", r.ID)

	docs, err := h.dbHandler.DocumentGetsByRagID(ctx, id)
	if err != nil {
		log.Errorf("Could not get documents for rag. err: %v", err)
		return nil, fmt.Errorf("could not get documents for rag: %w", err)
	}
	log.Debugf("Retrieved %d documents for rag. rag_id: %s", len(docs), id)

	r.Status = computeRagStatus(docs)
	r.Sources = buildSources(docs)

	return r, nil
}
```

**Step 3: Update `RagList` to enrich via batch fetch**

```go
func (h *ragHandler) RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RagList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	rags, err := h.dbHandler.RagList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list rags. err: %v", err)
		return nil, fmt.Errorf("could not list rags: %w", err)
	}
	log.Debugf("Listed rags. count: %d", len(rags))

	if len(rags) == 0 {
		return rags, nil
	}

	ragIDs := make([]uuid.UUID, len(rags))
	for i, r := range rags {
		ragIDs[i] = r.ID
	}

	docsMap, err := h.dbHandler.DocumentGetsByRagIDs(ctx, ragIDs)
	if err != nil {
		log.Errorf("Could not batch fetch documents for rags. err: %v", err)
		return rags, nil
	}

	for _, r := range rags {
		docs := docsMap[r.ID]
		r.Status = computeRagStatus(docs)
		r.Sources = buildSources(docs)
	}

	return rags, nil
}
```

**Step 3.5: Add `RagAddSources` to `rag.go`**

```go
func (h *ragHandler) RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "RagAddSources",
		"rag_id": ragID,
	})

	r, err := h.dbHandler.RagGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return nil, fmt.Errorf("could not get rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Retrieved rag for adding sources. rag_id: %s", r.ID)

	h.createDocumentsForSources(ctx, r.CustomerID, r.ID, storageFileIDs, sourceURLs)

	return h.RagGet(ctx, ragID)
}
```

**Step 4: Add helper functions to `rag.go`**

**IMPORTANT:** The functions below reference `document.StatusPending`, `document.StatusProcessing`, etc. The existing `rag.go` only imports `"monorepo/bin-rag-manager/models/rag"`. You MUST add `"monorepo/bin-rag-manager/models/document"` to the import block in `rag.go`.

```go
// createDocumentsForSources creates documents for each file ID and URL, then triggers ingestion.
// Uses request ctx for DB writes; ingestion goroutines use context.Background().
func (h *ragHandler) createDocumentsForSources(ctx context.Context, customerID, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "createDocumentsForSources",
		"rag_id": ragID,
	})
	for _, fileID := range storageFileIDs {
		doc, err := h.documentCreateInternal(ctx, customerID, ragID, fileID, "")
		if err != nil {
			log.Errorf("Could not create document for file_id %s: %v", fileID, err)
			continue
		}
		go h.documentIngest(doc)
	}

	for _, url := range sourceURLs {
		doc, err := h.documentCreateInternal(ctx, customerID, ragID, uuid.Nil, url)
		if err != nil {
			log.Errorf("Could not create document for url %s: %v", url, err)
			continue
		}
		go h.documentIngest(doc)
	}
}

// computeRagStatus derives RAG status from its documents.
func computeRagStatus(docs []*document.Document) document.Status {
	if len(docs) == 0 {
		return document.StatusPending
	}

	hasPending := false
	hasProcessing := false
	hasError := false
	allReady := true

	for _, d := range docs {
		switch d.Status {
		case document.StatusPending:
			hasPending = true
			allReady = false
		case document.StatusProcessing:
			hasProcessing = true
			allReady = false
		case document.StatusError:
			hasError = true
			allReady = false
		case document.StatusReady:
			// no-op
		}
	}

	if allReady {
		return document.StatusReady
	}
	if hasPending || hasProcessing {
		return document.StatusProcessing
	}
	if hasError {
		return document.StatusError
	}
	return document.StatusPending
}

// buildSources converts documents to Source structs for the RAG response.
func buildSources(docs []*document.Document) []rag.Source {
	sources := make([]rag.Source, 0, len(docs))
	for _, d := range docs {
		s := rag.Source{
			Status:        d.Status,
			StatusMessage: d.StatusMessage,
		}
		if d.StorageFileID != uuid.Nil {
			fileID := d.StorageFileID
			s.StorageFileID = &fileID
		}
		if d.SourceURL != "" {
			s.SourceURL = d.SourceURL
		}
		sources = append(sources, s)
	}
	return sources
}
```

**Step 5: Update `document.go` — Internal document creation and ingestion pipeline**

**IMPORTANT:** Do NOT replace the entire file. **Keep the existing `DocumentGet` and `DocumentList` methods** — they remain in the public interface. Remove the old `DocumentCreate` and `DocumentDelete` methods, then add the new internal functions below. Merge the import block (add new imports to the existing ones).

Add these imports to the existing import block (keep existing `"context"`, `"fmt"`, `"github.com/gofrs/uuid"`, `"github.com/sirupsen/logrus"`, `"monorepo/bin-rag-manager/models/document"`):

```go
// Add to existing imports:
"io"
"net/http"
"net/url"
"os"
"path"
"time"

dbchunk "monorepo/bin-rag-manager/models/chunk"
"monorepo/bin-rag-manager/pkg/chunker"
```

Add the following constants and functions (after removing old `DocumentCreate` and `DocumentDelete`):

const (
	maxFileSize       = 50 * 1024 * 1024 // 50 MB
	maxRetryCount     = 3
	maxTokensPerChunk = 512
	heartbeatInterval = 10 // Update heartbeat every 10 chunks
)

// documentCreateInternal creates a document internally (not exposed via API).
// Uses the request ctx for DB operations (these must complete before the goroutine launches).
// NOTE: All methods in this file use `logrus.WithFields(...)` per-function (matching existing pattern).
func (h *ragHandler) documentCreateInternal(ctx context.Context, customerID, ragID uuid.UUID, storageFileID uuid.UUID, sourceURL string) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "documentCreateInternal",
		"customer_id": customerID,
		"rag_id":      ragID,
	})
	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not create uuid: %w", err)
	}

	docType := document.DocTypeURL
	name := sourceURL
	if storageFileID != uuid.Nil {
		docType = document.DocTypeUploaded
		name = storageFileID.String()
	}

	d := &document.Document{
		ID:            id,
		CustomerID:    customerID,
		RagID:         ragID,
		Name:          name,
		DocType:       docType,
		StorageFileID: storageFileID,
		SourceURL:     sourceURL,
		Status:        document.StatusPending,
	}

	if err := h.dbHandler.DocumentCreate(ctx, d); err != nil {
		return nil, fmt.Errorf("could not create document: %w", err)
	}

	res, err := h.dbHandler.DocumentGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get created document: %w", err)
	}
	log.WithField("document", res).Debugf("Created document internally. document_id: %s", res.ID)

	return res, nil
}

// documentIngest is the core async ingestion pipeline.
// It runs in a goroutine with context.Background().
func (h *ragHandler) documentIngest(doc *document.Document) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "documentIngest",
		"document_id": doc.ID,
	})
	ctx := context.Background()

	// Step 1: Atomic claim
	claimed, err := h.dbHandler.DocumentClaimForProcessing(ctx, doc.ID)
	if err != nil {
		log.Debugf("Could not claim document for processing, likely already claimed. document_id: %s, err: %v", doc.ID, err)
		return
	}
	log.WithField("document", claimed).Debugf("Claimed document for processing. document_id: %s, retry_count: %d", claimed.ID, claimed.RetryCount)

	// Step 2: Acquire file
	tmpPath, filename, contentType, err := h.documentAcquireFile(ctx, claimed)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not acquire file: %w", err))
		return
	}
	defer func() { _ = os.Remove(tmpPath) }()
	log.Debugf("Acquired file. document_id: %s, filename: %s, tmpPath: %s", claimed.ID, filename, tmpPath)

	// Step 3: Detect format
	ext := chunker.DetectExtensionFromFilename(filename)
	if ext == "" {
		ext = chunker.DetectExtensionFromContentType(contentType)
	}
	c := chunker.GetChunkerByExtension(ext)
	log.Debugf("Detected format. document_id: %s, ext: %s", claimed.ID, ext)

	// Step 4: Chunk content
	chunks, err := c.Chunk(tmpPath, maxTokensPerChunk)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not chunk file: %w", err))
		return
	}
	if len(chunks) == 0 {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("file produced no chunks"))
		return
	}
	log.Debugf("Chunked file. document_id: %s, chunk_count: %d", claimed.ID, len(chunks))

	// Step 5: Update heartbeat after chunking
	if err := h.dbHandler.DocumentUpdateHeartbeat(ctx, claimed.ID); err != nil {
		log.Errorf("Could not update heartbeat after chunking. document_id: %s, err: %v", claimed.ID, err)
	}

	// Step 6: Generate embeddings
	texts := make([]string, len(chunks))
	for i, ch := range chunks {
		texts[i] = ch.Text
	}

	embeddings, err := h.embedTextsWithHeartbeat(ctx, claimed.ID, texts)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not generate embeddings: %w", err))
		return
	}
	log.Debugf("Generated embeddings. document_id: %s, embedding_count: %d", claimed.ID, len(embeddings))

	// Step 7: Store chunks
	dbChunks, err := h.buildDBChunks(claimed, chunks)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not build db chunks: %w", err))
		return
	}

	if err := h.dbHandler.ChunkCreateBatch(ctx, dbChunks, embeddings); err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not store chunks: %w", err))
		return
	}
	log.Debugf("Stored chunks. document_id: %s, chunk_count: %d", claimed.ID, len(dbChunks))

	// Step 8: Set status to ready
	if err := h.dbHandler.DocumentUpdate(ctx, claimed.ID, map[document.Field]any{
		document.FieldStatus: document.StatusReady,
	}); err != nil {
		log.Errorf("Could not set document to ready. document_id: %s, err: %v", claimed.ID, err)
		return
	}
	log.Infof("Document ingestion complete. document_id: %s, chunks: %d", claimed.ID, len(dbChunks))
}

// documentIngestFail handles ingestion failure with retry logic.
func (h *ragHandler) documentIngestFail(ctx context.Context, doc *document.Document, ingestErr error) {
	log := logrus.WithField("func", "documentIngestFail")
	log.Errorf("Document ingestion failed. document_id: %s, retry_count: %d, err: %v", doc.ID, doc.RetryCount, ingestErr)

	if doc.RetryCount >= maxRetryCount {
		// Terminal failure
		_ = h.dbHandler.DocumentUpdate(ctx, doc.ID, map[document.Field]any{
			document.FieldStatus:        document.StatusError,
			document.FieldStatusMessage: ingestErr.Error(),
		})
		return
	}

	// Reset to pending for retry
	_ = h.dbHandler.DocumentUpdate(ctx, doc.ID, map[document.Field]any{
		document.FieldStatus: document.StatusPending,
	})
}

// documentAcquireFile downloads the file to a temp file.
// Returns tmpPath, filename, contentType.
func (h *ragHandler) documentAcquireFile(ctx context.Context, doc *document.Document) (string, string, string, error) {
	if doc.StorageFileID != uuid.Nil {
		return h.documentDownloadGCS(ctx, doc.StorageFileID)
	}
	if doc.SourceURL != "" {
		return h.documentDownloadURL(ctx, doc.SourceURL)
	}
	return "", "", "", fmt.Errorf("document has no storage_file_id or source_url")
}

// documentDownloadGCS fetches file metadata from storage-manager and downloads from GCS.
func (h *ragHandler) documentDownloadGCS(ctx context.Context, storageFileID uuid.UUID) (string, string, string, error) {
	log := logrus.WithField("func", "documentDownloadGCS")
	file, err := h.reqHandler.StorageV1FileGet(ctx, storageFileID)
	if err != nil {
		return "", "", "", fmt.Errorf("could not get file metadata: %w", err)
	}
	log.WithField("file", file).Debugf("Retrieved storage file metadata. file_id: %s", storageFileID)

	if file.Filesize > maxFileSize {
		return "", "", "", fmt.Errorf("file size %d exceeds limit %d", file.Filesize, maxFileSize)
	}

	bucketName := file.BucketName
	if bucketName == "" {
		bucketName = h.bucketName
	}

	tmpPath, err := h.bucketReader.DownloadToTempFile(ctx, bucketName, file.Filepath)
	if err != nil {
		return "", "", "", fmt.Errorf("could not download from GCS: %w", err)
	}

	return tmpPath, file.Filename, "", nil
}

// documentDownloadURL downloads a URL to a temp file with size limit enforcement.
func (h *ragHandler) documentDownloadURL(ctx context.Context, sourceURL string) (string, string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("could not create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("could not fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	if resp.ContentLength > maxFileSize {
		return "", "", "", fmt.Errorf("content length %d exceeds limit %d", resp.ContentLength, maxFileSize)
	}

	tmpFile, err := os.CreateTemp("", "rag_url_*")
	if err != nil {
		return "", "", "", fmt.Errorf("could not create temp file: %w", err)
	}

	limitReader := io.LimitReader(resp.Body, maxFileSize+1)
	n, err := io.Copy(tmpFile, limitReader)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", "", "", fmt.Errorf("could not download URL: %w", err)
	}
	if n > maxFileSize {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", "", "", fmt.Errorf("download size %d exceeds limit %d", n, maxFileSize)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", "", "", fmt.Errorf("could not close temp file: %w", err)
	}

	// Extract filename from URL path
	parsedURL, _ := url.Parse(sourceURL)
	filename := ""
	if parsedURL != nil {
		filename = path.Base(parsedURL.Path)
	}

	contentType := resp.Header.Get("Content-Type")
	return tmpFile.Name(), filename, contentType, nil
}

// embedTextsWithHeartbeat generates embeddings in batches and updates heartbeat periodically.
// Uses EmbedTexts (RETRIEVAL_DOCUMENT task type) — correct for indexing document chunks.
// EmbedText uses RETRIEVAL_QUERY task type and is for query-time only.
func (h *ragHandler) embedTextsWithHeartbeat(ctx context.Context, docID uuid.UUID, texts []string) ([][]float32, error) {
	log := logrus.WithField("func", "embedTextsWithHeartbeat")
	allEmbeddings := make([][]float32, 0, len(texts))

	for i := 0; i < len(texts); i += heartbeatInterval {
		end := i + heartbeatInterval
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embs, err := h.embedder.EmbedTexts(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("could not embed chunks %d-%d: %w", i, end-1, err)
		}
		allEmbeddings = append(allEmbeddings, embs...)

		if err := h.dbHandler.DocumentUpdateHeartbeat(ctx, docID); err != nil {
			log.Errorf("Could not update heartbeat during embedding. document_id: %s, err: %v", docID, err)
		}
	}

	return allEmbeddings, nil
}

// buildDBChunks converts chunker Chunks to database Chunk models.
func (h *ragHandler) buildDBChunks(doc *document.Document, chunks []chunker.Chunk) ([]*dbchunk.Chunk, error) {
	dbChunks := make([]*dbchunk.Chunk, 0, len(chunks))

	for i, ch := range chunks {
		id, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("could not create uuid: %w", err)
		}
		dbChunks = append(dbChunks, &dbchunk.Chunk{
			ID:           id,
			DocumentID:   doc.ID,
			RagID:        doc.RagID,
			CustomerID:   doc.CustomerID,
			ChunkIndex:   i,
			Text:         ch.Text,
			SectionTitle: ch.SectionTitle,
			TokenCount:   len(ch.Text) / 4,
		})
	}

	return dbChunks, nil
}
```

**Important:** The import for chunk model should use the existing alias pattern. Check the existing imports in `raghandler/document.go` — the chunk model is at `monorepo/bin-rag-manager/models/chunk`. Import it as `dbchunk "monorepo/bin-rag-manager/models/chunk"` to avoid collision with the `chunker` package.

**Step 6: Remove old `DocumentCreate` and `DocumentDelete` from document.go**

Remove the `DocumentCreate` method (the one that was public and took `docType` parameter). Replace with the internal version above.

Remove the `DocumentDelete` method from `document.go`. Document deletion is now handled only through `RagDelete` (cascade).

Keep `DocumentGet` and `DocumentList` as they are — these remain in the public interface for read-only access.

**Step 7: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 8: Commit**

```bash
git add bin-rag-manager/pkg/raghandler/rag.go bin-rag-manager/pkg/raghandler/document.go \
        bin-rag-manager/pkg/raghandler/main.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Implement document ingestion pipeline with GCS and URL download
- bin-rag-manager: Update RagCreate to accept file IDs and URLs, create documents internally
- bin-rag-manager: Add RagAddSources and RagGetWithSources with status computation"
```

---

### Task 12: RagHandler — Startup Sweep and Periodic Ticker

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/document.go`

**Step 1: Add `DocumentIngestPendingAll`**

Add to the end of `document.go`:

```go
// DocumentIngestPendingAll resets stale processing documents and triggers pending ones.
// Called on startup before accepting requests.
// Uses 2-minute threshold so documents actively being processed by other pods are not reset.
func (h *ragHandler) DocumentIngestPendingAll(ctx context.Context) {
	log := logrus.WithField("func", "DocumentIngestPendingAll")
	// Reset stale processing documents (2min threshold avoids resetting docs actively processed by other pods)
	if err := h.dbHandler.DocumentResetStaleToPending(ctx, 2*time.Minute); err != nil {
		log.Errorf("Could not reset processing documents on startup: %v", err)
	}

	// Trigger pending documents
	docs, err := h.dbHandler.DocumentGetPending(ctx)
	if err != nil {
		log.Errorf("Could not get pending documents on startup: %v", err)
		return
	}

	log.Infof("Startup sweep: found %d pending documents to process", len(docs))
	for _, doc := range docs {
		go h.documentIngest(doc)
	}
}

// RunIngestionTicker periodically checks for stale and pending documents.
func (h *ragHandler) RunIngestionTicker(ctx context.Context, interval time.Duration) {
	log := logrus.WithField("func", "RunIngestionTicker")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Ingestion ticker stopped")
			return
		case <-ticker.C:
			h.ingestionTickerRun(ctx, interval)
		}
	}
}

func (h *ragHandler) ingestionTickerRun(ctx context.Context, threshold time.Duration) {
	log := logrus.WithField("func", "ingestionTickerRun")
	// Reset stale processing documents
	if err := h.dbHandler.DocumentResetStaleToPending(ctx, threshold); err != nil {
		log.Errorf("Ticker: could not reset stale documents: %v", err)
	}

	// Trigger pending documents
	docs, err := h.dbHandler.DocumentGetPending(ctx)
	if err != nil {
		log.Errorf("Ticker: could not get pending documents: %v", err)
		return
	}

	if len(docs) > 0 {
		log.Infof("Ticker: found %d pending documents to process", len(docs))
	}
	for _, doc := range docs {
		go h.documentIngest(doc)
	}
}
```

**Step 2: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```bash
git add bin-rag-manager/pkg/raghandler/document.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Add startup sweep and periodic ticker for stuck document recovery"
```

---

### Task 13: ListenHandler — Update Routes for New API

**Files:**
- Modify: `bin-rag-manager/pkg/listenhandler/main.go`
- Modify: `bin-rag-manager/pkg/listenhandler/main_test.go`
- Modify: `bin-rag-manager/pkg/listenhandler/v1_rags.go`
- Modify: `bin-rag-manager/pkg/listenhandler/v1_documents.go`

**Step 1: Add new route regex for sources endpoint**

In `main.go`, add:

```go
regV1RagsIDSources = regexp.MustCompile(`^/v1/rags/` + regUUID + `/sources(\?.*)?$`)
```

**Step 2: Add route matching in `processRequest`**

**CRITICAL (M3 fix): The `regV1RagsIDSources` match MUST come BEFORE the `regV1RagsID` match**, because `/v1/rags/<uuid>/sources` also matches the `/v1/rags/<uuid>` pattern. More specific routes must be checked first.

```go
// MUST come BEFORE regV1RagsID cases — more specific route first
case regV1RagsIDSources.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
    response, err = h.processV1RagsIDSourcesPost(ctx, m)
    requestType = "/v1/rags/<rag-id>/sources"
```

**IMPORTANT:** This case uses the same flat `case regex.MatchString(m.URI) && m.Method == ...` pattern as all other routes. Do NOT use nested `switch` on method. The variables are `m.URI` and `m.Method` (where `m *sock.Request`), and results go into `response` and `err`.

**Step 3: Update `processV1RagsPost` in `v1_rags.go`**

Update the request struct and handler to accept `storage_file_ids` and `source_urls`. **Match existing pattern:** parameter is `m *sock.Request`, use `log := logrus.WithFields(...)`, return `simpleResponse(4xx), nil` for errors (NOT `fmt.Errorf`).

```go
func (h *listenHandler) processV1RagsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsPost",
	})

	var reqData struct {
		CustomerID     uuid.UUID   `json:"customer_id"`
		Name           string      `json:"name"`
		Description    string      `json:"description"`
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}

	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	if reqData.CustomerID == uuid.Nil {
		log.Errorf("Customer ID is required.")
		return simpleResponse(400), nil
	}

	if reqData.Name == "" {
		log.Errorf("Name is required.")
		return simpleResponse(400), nil
	}

	if len(reqData.StorageFileIDs) == 0 && len(reqData.SourceURLs) == 0 {
		log.Errorf("At least one storage_file_ids or source_urls is required.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagCreate(ctx, reqData.CustomerID, reqData.Name, reqData.Description, reqData.StorageFileIDs, reqData.SourceURLs)
	if err != nil {
		log.Errorf("Could not create rag. err: %v", err)
		return simpleResponse(500), nil
	}

	return jsonResponse(200, r), nil
}
```

**Step 4: Update `processV1RagsIDGet` to use enriched RagGet**

`RagGet` already returns `*rag.Rag` with transient Status/Sources populated. No changes needed to the existing handler IF it already calls `h.ragHandler.RagGet()` and marshals the result. Verify that the existing handler does this — if it does, no change needed for this handler.

**Step 5: Add `processV1RagsIDSourcesPost` to `v1_rags.go`**

```go
func (h *listenHandler) processV1RagsIDSourcesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDSourcesPost",
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	ragID := uuid.FromStringOrNil(uriItems[3])
	if ragID == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	var reqData struct {
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}

	if err := json.Unmarshal(m.Data, &reqData); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	if len(reqData.StorageFileIDs) == 0 && len(reqData.SourceURLs) == 0 {
		log.Errorf("At least one storage_file_ids or source_urls is required.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagAddSources(ctx, ragID, reqData.StorageFileIDs, reqData.SourceURLs)
	if err != nil {
		log.Errorf("Could not add sources. err: %v", err)
		return simpleResponse(500), nil
	}

	return jsonResponse(200, r), nil
}
```

**Step 6: Update `v1_documents.go` — Remove POST and DELETE handlers**

Remove `processV1DocumentsPost` and `processV1DocumentsIDDelete` functions. Keep `processV1DocumentsGet` and `processV1DocumentsIDGet` (read-only).

In `processRequest`, remove the `POST` case for `regV1Documents` and the `DELETE` case for `regV1DocumentsID`.

**Step 7: Update `listenhandler/main_test.go` manual mock**

The file defines a manual `mockRagHandlerForListen` struct that implements the `RagHandler` interface. After Task 10's interface changes, this mock will fail to compile. Update it:

1. **Fix `RagCreate` signature** — change from `(ctx, uuid.UUID, string, string)` to `(ctx, uuid.UUID, string, string, []uuid.UUID, []string)`:
```go
func (m *mockRagHandlerForListen) RagCreate(_ context.Context, _ uuid.UUID, _, _ string, _ []uuid.UUID, _ []string) (*rag.Rag, error) {
    return nil, fmt.Errorf("not implemented")
}
```

2. **Add 3 new interface methods** (missing from the mock):
```go
func (m *mockRagHandlerForListen) RagAddSources(_ context.Context, _ uuid.UUID, _ []uuid.UUID, _ []string) (*rag.Rag, error) {
    return nil, fmt.Errorf("not implemented")
}
func (m *mockRagHandlerForListen) DocumentIngestPendingAll(_ context.Context) {}
func (m *mockRagHandlerForListen) RunIngestionTicker(_ context.Context, _ time.Duration) {}
```

3. **Add `"time"` import** to the test file (needed for `time.Duration` in `RunIngestionTicker`).

4. Optionally remove `DocumentCreate` and `DocumentDelete` methods from the mock (they are dead code after interface removal, but won't cause compilation errors if left).

**Step 8: Run verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 9: Commit**

```bash
git add bin-rag-manager/pkg/listenhandler/
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Update RagCreate handler to accept file IDs and URLs
- bin-rag-manager: Add POST /rags/{id}/sources endpoint
- bin-rag-manager: Update RagGet to return WebhookMessage with status and sources
- bin-rag-manager: Remove document POST and DELETE handlers (read-only now)
- bin-rag-manager: Update listenhandler manual mock for new RagHandler interface"
```

---

### Task 14: Main Wiring — RequestHandler, GCS, Sweep, Ticker

**Files:**
- Modify: `bin-rag-manager/cmd/rag-manager/main.go`

**Step 1: Update `runService` to wire new dependencies**

In `cmd/rag-manager/main.go`, update `runService()`. **IMPORTANT:** Keep the existing function signature (`func runService(cfg config.Config) error`), `sockHandler.Connect()`, `db.Ping()`, and `defer db.Close()`. Only add the new wiring — do not drop existing setup.

```go
func runService(cfg config.Config) error {
	log := logrus.WithField("func", "runService")

	ctx := context.Background()

	// RabbitMQ connection
	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, cfg.RabbitMQAddress)
	sockHandler.Connect()

	// PostgreSQL connection
	db, err := sql.Open("postgres", cfg.PostgreSQLDSN)
	if err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("could not ping PostgreSQL: %w", err)
	}
	log.Info("Connected to PostgreSQL")

	// Request handler — NewRequestHandler takes (sockHandler, publisherServiceName)
	reqHandler := requesthandler.NewRequestHandler(sockHandler, commonoutline.ServiceNameRagManager)

	// GCS client
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create GCS client: %w", err)
	}

	// Bucket reader
	br := bucketreader.NewBucketReader(gcsClient)

	dbH := dbhandler.NewHandler(db)

	emb, err := embedder.NewGoogleEmbedder(ctx, cfg.GoogleCloudProject, cfg.GoogleCloudLocation, cfg.GoogleEmbeddingModel)
	if err != nil {
		return fmt.Errorf("could not create embedder: %w", err)
	}

	ragH := raghandler.NewRagHandler(emb, dbH, reqHandler, br, cfg.GCPBucketNameMedia)

	// Startup sweep: re-process pending documents
	ragH.DocumentIngestPendingAll(ctx)

	// Start periodic ticker
	go ragH.RunIngestionTicker(ctx, 5*time.Minute)

	if err := runListen(sockHandler, ragH); err != nil {
		return err
	}

	// Block until shutdown signal — keeps DB connection alive for request handlers
	<-chDone

	return nil
}
```

Add imports — **only add imports not already present**. `commonoutline` and `sock` are already imported in the existing `main.go`. Only add:

```go
import (
	"cloud.google.com/go/storage"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-rag-manager/pkg/bucketreader"
)
```

Do NOT re-add `commonoutline "monorepo/bin-common-handler/models/outline"` or `"monorepo/bin-common-handler/models/sock"` — they are already imported (see existing lines 13-14 of `main.go`).

Note: Check how `requesthandler.NewRequestHandler` is called in other services (e.g., `bin-flow-manager/cmd/flow-manager/main.go`). The signature typically takes `(sockHandler, publisherServiceName)`. Match the existing pattern.

**Step 2: Run full verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```bash
git add bin-rag-manager/cmd/rag-manager/main.go bin-rag-manager/go.mod bin-rag-manager/go.sum
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-rag-manager: Wire requesthandler, GCS client, bucket reader, startup sweep, and ticker"
```

---

### Task 15: common-handler — Update RPC Calls

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/rag_rags.go`
- Modify: `bin-common-handler/pkg/requesthandler/rag_documents.go`

**Step 1: Update `RagV1RagCreate` signature and request body**

Add `storageFileIDs []uuid.UUID` and `sourceURLs []string` parameters:

```go
func (r *requestHandler) RagV1RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.Rag, error) {
	m, err := json.Marshal(struct {
		CustomerID     uuid.UUID   `json:"customer_id"`
		Name           string      `json:"name"`
		Description    string      `json:"description"`
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}{
		CustomerID:     customerID,
		Name:           name,
		Description:    description,
		StorageFileIDs: storageFileIDs,
		SourceURLs:     sourceURLs,
	})
	// ... rest stays the same
}
```

**Step 2: Add `RagV1RagAddSources`**

**IMPORTANT (C1 fix):** `sendRequestRag` takes **8 parameters**: `(ctx, uri, method, dataType, timeout, delay, contentType, data)`. Match the existing call pattern from `RagV1RagCreate`.

```go
func (r *requestHandler) RagV1RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s/sources", ragID)

	m, err := json.Marshal(struct {
		StorageFileIDs []uuid.UUID `json:"storage_file_ids"`
		SourceURLs     []string    `json:"source_urls"`
	}{
		StorageFileIDs: storageFileIDs,
		SourceURLs:     sourceURLs,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/rags.sources", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

**Step 3: `RagV1RagGet` return type stays `*rmrag.Rag`**

No change needed. The transient fields (Status, Sources) are serialized in JSON by rag-manager's listenhandler and deserialized into `*rmrag.Rag` by the requesthandler. Since the Rag struct now has `Status` and `Sources` fields (json tags, no db tags), JSON unmarshaling populates them automatically.

**Step 4: Update `rag_documents.go` — Remove `RagV1DocumentCreate` and `RagV1DocumentDelete`**

Remove these two methods. Keep `RagV1DocumentGet` and `RagV1DocumentGets` (read-only).

**Step 5: Update the RequestHandler interface**

In `bin-common-handler/pkg/requesthandler/main.go`, find the RAG section of the `RequestHandler` interface (around lines 1316-1329) and make these exact changes:

```go
// Replace existing:
RagV1RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rmrag.Rag, error)
// With:
RagV1RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.Rag, error)

// Add new method (after RagV1RagDelete):
RagV1RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.Rag, error)

// Remove these two lines:
RagV1DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType rmdocument.DocType, sourceURL string, storageFileID uuid.UUID) (*rmdocument.Document, error)
RagV1DocumentDelete(ctx context.Context, id uuid.UUID) error

// Keep these unchanged:
RagV1RagGet(ctx context.Context, id uuid.UUID) (*rmrag.Rag, error)
RagV1RagGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmrag.Field]any) ([]*rmrag.Rag, error)
RagV1RagUpdate(ctx context.Context, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.Rag, error)
RagV1RagDelete(ctx context.Context, id uuid.UUID) error
RagV1DocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error)
RagV1DocumentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmdocument.Field]any) ([]*rmdocument.Document, error)
```

**Step 6: Run verification for common-handler**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-common-handler/pkg/requesthandler/rag_rags.go \
        bin-common-handler/pkg/requesthandler/rag_documents.go \
        bin-common-handler/pkg/requesthandler/main.go
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-common-handler: Update RagV1RagCreate to accept file IDs and URLs
- bin-common-handler: Add RagV1RagAddSources RPC method
- bin-common-handler: Update RagV1RagGet to return WebhookMessage
- bin-common-handler: Remove RagV1DocumentCreate and RagV1DocumentDelete (documents now internal)"
```

---

### Task 16: api-manager — Update Service Handler and Server

**PREREQUISITE:** Task 17 (OpenAPI spec update) MUST be completed and `go generate ./...` run in `bin-openapi-manager` and `bin-api-manager` BEFORE this task. The server handler code references generated types (`PostRagsJSONBody.StorageFileIds`, `PostRagsIdSourcesJSONBody`, etc.) that don't exist until the OpenAPI spec is updated and code is regenerated.

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/rag.go`
- Modify: `bin-api-manager/pkg/servicehandler/rag_document.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go`
- Modify: `bin-api-manager/server/rags.go`
- Modify: `bin-api-manager/server/rags_test.go`
- Modify: `bin-api-manager/server/rag_documents.go`
- Modify: `bin-api-manager/server/rag_documents_test.go`

**Step 1: Update `servicehandler/main.go` interface**

Update the RAG-related methods:

```go
// Remove:
RagDocumentCreate(...)
RagDocumentDelete(...)

// Update RagCreate:
RagCreate(ctx context.Context, a *amagent.Agent, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error)

// Add:
RagAddSources(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error)
```

Note: `RagGet` flow is unchanged — `ragGet()` returns `*rmrag.Rag` (with transient Status/Sources from JSON), then `.ConvertWebhookMessage()` copies them through. No type changes needed in existing code.

**Step 2: Update `servicehandler/rag.go`**

Update `RagCreate` to pass file IDs and URLs. **IMPORTANT (C5 fix):** `hasPermission` takes **4 args**: `(ctx, agent, customerID, permission)`. Match the existing pattern in `rag.go`.

```go
func (h *serviceHandler) RagCreate(ctx context.Context, a *amagent.Agent, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": a.CustomerID,
		"name":        name,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	log.Debug("Creating a new rag.")
	tmp, err := h.reqHandler.RagV1RagCreate(ctx, a.CustomerID, name, description, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not create a new rag. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

Since `RagV1RagCreate` returns `*rmrag.Rag` with transient Status/Sources populated (via JSON from the enriched response), `.ConvertWebhookMessage()` copies them through naturally. No type mismatch.

Add `RagAddSources`:

```go
func (h *serviceHandler) RagAddSources(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagAddSources",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
	})
	log.Debug("Adding sources to rag.")

	// Verify RAG exists and belongs to this customer
	tmp, err := h.ragGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	r, err := h.reqHandler.RagV1RagAddSources(ctx, ragID, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not add sources. err: %v", err)
		return nil, err
	}

	res := r.ConvertWebhookMessage()
	return res, nil
}
```

**Step 3: Update `servicehandler/rag_document.go`**

Remove `RagDocumentCreate` and `RagDocumentDelete` methods. Keep `RagDocumentGet` and `RagDocumentGets` (read-only).

**Step 4: Update `server/rags.go`**

**IMPORTANT:** The existing file uses `h *server` receiver (not `h *handler`) and `c.BindJSON` (not `c.ShouldBindJSON`). Match the existing pattern exactly. Variables are `a` (agent), not `agent`, and context is `c.Request.Context()`.

Update `PostRags` to extract file IDs and URLs from the request:

```go
func (h *server) PostRags(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRags",
		"request_address": c.ClientIP(),
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostRagsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	storageFileIDs := []uuid.UUID{}
	if req.StorageFileIds != nil {
		for _, id := range *req.StorageFileIds {
			storageFileIDs = append(storageFileIDs, uuid.FromStringOrNil(id.String()))
		}
	}

	sourceURLs := []string{}
	if req.SourceUrls != nil {
		sourceURLs = *req.SourceUrls
	}

	res, err := h.serviceHandler.RagCreate(c.Request.Context(), &a, req.Name, description, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not create data. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

Add `PostRagsIdSources` handler (follow the same pattern — `h *server`, `c.BindJSON`, `c.Request.Context()`, `&a`):

```go
func (h *server) PostRagsIdSources(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRagsIdSources",
		"request_address": c.ClientIP(),
		"target_id":       id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target, err := uuid.FromString(id.String())
	if err != nil {
		log.Errorf("Invalid ID format. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PostRagsIdSourcesJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	storageFileIDs := []uuid.UUID{}
	if req.StorageFileIds != nil {
		for _, fid := range *req.StorageFileIds {
			storageFileIDs = append(storageFileIDs, uuid.FromStringOrNil(fid.String()))
		}
	}

	sourceURLs := []string{}
	if req.SourceUrls != nil {
		sourceURLs = *req.SourceUrls
	}

	res, err := h.serviceHandler.RagAddSources(c.Request.Context(), &a, target, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not add sources. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

**Note:** The exact generated type names (`PostRagsIdSourcesJSONBody`, field names like `StorageFileIds`) depend on the OpenAPI spec (Task 17). Adjust to match what `go generate` produces.

**Step 5: Update `server/rag_documents.go`**

Remove `PostRagDocuments` and `DeleteRagDocumentsId` handler methods. Keep `GetRagDocuments` and `GetRagDocumentsId` (read-only).

**Step 6: Fix test files (C4 fix)**

Update ALL test files that reference changed or removed methods:

1. `bin-api-manager/pkg/servicehandler/rag_document_test.go` — remove test cases that call mocks for `RagV1DocumentCreate` and `RagV1DocumentDelete` (these RPC methods no longer exist). Also update `rag_test.go` if it references the old `RagCreate` signature (3 args → 5 args).

2. `bin-api-manager/server/rag_documents_test.go` — remove `Test_PostRagDocuments` and `Test_DeleteRagDocumentsId` (these call `mockSvc.EXPECT().RagDocumentCreate(...)` and `mockSvc.EXPECT().RagDocumentDelete(...)` which no longer exist in the `ServiceHandler` interface after Step 1). Keep `Test_GetRagDocuments` and `Test_GetRagDocumentsId` (read-only endpoints remain).

3. `bin-api-manager/server/rags_test.go` — update `Test_PostRags` mock call from 4 args `(ctx, agent, name, description)` to 6 args `(ctx, agent, name, description, storageFileIDs, sourceURLs)`. Update the test `reqBody` JSON to include `storage_file_ids` and `source_urls` fields.

**Step 7: Find ALL callers of `RagV1RagCreate` across monorepo (H3 fix)**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline
grep -r "RagV1RagCreate" --include="*.go" | grep -v vendor | grep -v mock_
```

The `RagV1RagCreate` signature changed from `(ctx, customerID, name, description)` to `(ctx, customerID, name, description, storageFileIDs, sourceURLs)`. Update ALL callers found — not just api-manager.

**Step 8: Run verification for api-manager**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 9: Commit**

```bash
git add bin-api-manager/pkg/servicehandler/ bin-api-manager/server/
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-api-manager: Update RagCreate to accept file IDs and URLs
- bin-api-manager: Add RagAddSources endpoint handler
- bin-api-manager: Remove document create and delete handlers (documents now internal)
- bin-api-manager: Fix tests for removed RPC methods"
```

---

### Task 17: OpenAPI — Update Schemas and Endpoints

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml`
- Modify: OpenAPI path files for rags and rag-documents

**IMPORTANT:** Read `bin-openapi-manager/CLAUDE.md` first for the AI-Native Specification Rules before modifying any OpenAPI schema.

**Step 1: Update RAG create request schema**

Add `storage_file_ids` (array of UUIDs) and `source_urls` (array of strings) to the `PostRagsJSONBody` / RAG create request schema. Make at least one required.

**Step 2: Update RAG response schema (`RagManagerRag`)**

Add:
- `status` (reference to `RagManagerRagDocumentStatus`)
- `sources` (array of `RagManagerRagSource`)

Add new schema `RagManagerRagSource`:
```yaml
RagManagerRagSource:
  type: object
  properties:
    storage_file_id:
      type: string
      format: uuid
    source_url:
      type: string
      format: uri
    status:
      $ref: '#/components/schemas/RagManagerRagDocumentStatus'
    status_message:
      type: string
```

**Step 3: Add POST /rags/{id}/sources endpoint**

Create a new path file or add to existing rags path. Request body accepts `storage_file_ids` and `source_urls`. Response is the updated RAG.

**Step 4: Remove POST /rag-documents and DELETE /rag-documents/{id} endpoints**

Remove these from the path files. Keep GET /rag-documents and GET /rag-documents/{id} (read-only).

**Step 5: Regenerate OpenAPI types**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-openapi-manager
go generate ./...
```

**Step 6: Regenerate api-manager server code**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-api-manager
go generate ./...
```

**Step 7: Run verification for both**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 8: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-openapi-manager: Update RAG create request schema with storage_file_ids and source_urls
- bin-openapi-manager: Add status and sources to RAG response schema
- bin-openapi-manager: Add POST /rags/{id}/sources endpoint
- bin-openapi-manager: Remove POST /rag-documents and DELETE /rag-documents/{id} (documents now internal)
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 18: RST Documentation Updates

**Files:**
- Modify: RST files in `bin-api-manager/docsdev/source/` for RAG resource docs

**Step 1: Update RAG API documentation**

Update or create RST documentation for the RAG resource covering:
- Updated `POST /rags` request body (add `storage_file_ids`, `source_urls`)
- New `POST /rags/{id}/sources` endpoint
- Updated RAG response struct (add `status`, `sources`)
- Removed `POST /rag-documents` and `DELETE /rag-documents/{id}`
- Updated document endpoints as read-only

Follow the AI-Native RST Writing Guidelines in `bin-api-manager/CLAUDE.md`.

**Step 2: Rebuild HTML**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-api-manager/docsdev
rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline
git add bin-api-manager/docsdev/source/
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- bin-api-manager: Update RST docs for RAG ingestion pipeline API changes"
```

---

### Task 19: Final Verification and Cleanup

**Step 1: Run full verification for all changed services**

```bash
# rag-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-rag-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# common-handler
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-common-handler
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# api-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# openapi-manager
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify no other services import the changed requesthandler methods**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Implement-rag-document-ingestion-pipeline
grep -r "RagV1DocumentCreate\|RagV1DocumentDelete\|RagV1RagCreate" --include="*.go" | grep -v vendor | grep -v mock_
```

If any other services call removed methods or the old `RagV1RagCreate` signature, they need updating too.

**Step 3: Final commit (if any cleanup needed)**

```bash
git add -A
git commit -m "NOJIRA-Implement-rag-document-ingestion-pipeline

- Final verification and cleanup"
```
