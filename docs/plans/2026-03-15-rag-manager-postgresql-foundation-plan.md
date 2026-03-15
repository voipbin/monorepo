# RAG Manager PostgreSQL Foundation — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace bin-rag-manager's in-memory store with PostgreSQL + pgvector, add model structs, and create Alembic migrations for the new tables.

**Architecture:** Add a PostgreSQL-specific `dbhandler` package to bin-rag-manager (isolated from the monorepo's MySQL layer). Create proper model structs following monorepo conventions. Refactor raghandler to use the database instead of the in-memory store. Keep existing chunker, embedder, and generator packages mostly unchanged.

**Tech Stack:** Go, PostgreSQL 17, pgvector, `github.com/lib/pq`, `github.com/Masterminds/squirrel`, Alembic (Python) for migrations.

**Design Doc:** `docs/plans/2026-03-15-rag-manager-multi-tenant-database-design.md`

---

### Task 1: Add PostgreSQL Dependencies

**Files:**
- Modify: `bin-rag-manager/go.mod`

**Step 1: Add PostgreSQL driver and squirrel to go.mod**

```bash
cd bin-rag-manager
go get github.com/lib/pq
go get github.com/Masterminds/squirrel
go get github.com/gofrs/uuid
```

**Step 2: Run go mod tidy**

```bash
go mod tidy
```

**Step 3: Commit**

```bash
git add bin-rag-manager/go.mod bin-rag-manager/go.sum
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add PostgreSQL driver (lib/pq), squirrel query builder, and uuid dependencies"
```

---

### Task 2: Add PostgreSQL Config

**Files:**
- Modify: `bin-rag-manager/internal/config/config.go`
- Modify: `bin-rag-manager/cmd/rag-manager/main.go`

**Step 1: Add PostgreSQL DSN to Config struct and bindings**

In `config.go`, add to the `Config` struct:

```go
// PostgreSQL
PostgreSQLDSN string
```

In `Bootstrap()`, add to the flags:

```go
f.String("postgresql_dsn", "", "PostgreSQL connection string")
```

Add to the bindings map:

```go
"postgresql_dsn": "POSTGRESQL_DSN",
```

In `LoadGlobalConfig()`, add:

```go
PostgreSQLDSN: viper.GetString("postgresql_dsn"),
```

In `InitConfig()`, add the flag binding and viper binding for `postgresql_dsn`, and add to the globalConfig struct.

**Step 2: Add PostgreSQL flag to main.go init()**

In `cmd/rag-manager/main.go` `init()`, add:

```go
rootCmd.Flags().String("postgresql_dsn", "", "PostgreSQL connection string")
```

**Step 3: Run tests**

```bash
cd bin-rag-manager && go test ./internal/config/...
```

**Step 4: Commit**

```bash
git add bin-rag-manager/internal/config/config.go bin-rag-manager/cmd/rag-manager/main.go
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add PostgreSQL DSN configuration parameter"
```

---

### Task 3: Create RAG Model

**Files:**
- Create: `bin-rag-manager/models/rag/main.go`
- Create: `bin-rag-manager/models/rag/field.go`
- Create: `bin-rag-manager/models/rag/webhook.go`

**Step 1: Create the RAG model struct**

File: `bin-rag-manager/models/rag/main.go`

```go
package rag

import (
	"time"

	"github.com/gofrs/uuid"
)

// Rag represents a knowledge base container
type Rag struct {
	ID          uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	CustomerID  uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	Name        string     `json:"name,omitempty" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	TMCreate    *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate    *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete    *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

**Step 2: Create the Field type**

File: `bin-rag-manager/models/rag/field.go`

```go
package rag

// Field defines the type for rag field names
type Field string

// List of rag fields
const (
	FieldID          Field = "id"
	FieldCustomerID  Field = "customer_id"
	FieldName        Field = "name"
	FieldDescription Field = "description"
	FieldTMCreate    Field = "tm_create"
	FieldTMUpdate    Field = "tm_update"
	FieldTMDelete    Field = "tm_delete"
)
```

**Step 3: Create the WebhookMessage**

File: `bin-rag-manager/models/rag/webhook.go`

```go
package rag

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Rag
type WebhookMessage struct {
	ID          uuid.UUID  `json:"id,omitempty"`
	CustomerID  uuid.UUID  `json:"customer_id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	TMCreate    *time.Time `json:"tm_create,omitempty"`
	TMUpdate    *time.Time `json:"tm_update,omitempty"`
	TMDelete    *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts the internal Rag to the external representation
func (r *Rag) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:          r.ID,
		CustomerID:  r.CustomerID,
		Name:        r.Name,
		Description: r.Description,
		TMCreate:    r.TMCreate,
		TMUpdate:    r.TMUpdate,
		TMDelete:    r.TMDelete,
	}
}
```

**Step 4: Verify it compiles**

```bash
cd bin-rag-manager && go build ./models/rag/...
```

**Step 5: Commit**

```bash
git add bin-rag-manager/models/rag/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add Rag model with Field type and WebhookMessage conversion"
```

---

### Task 4: Create Document Model

**Files:**
- Create: `bin-rag-manager/models/document/main.go`
- Create: `bin-rag-manager/models/document/field.go`
- Create: `bin-rag-manager/models/document/webhook.go`

**Step 1: Create the Document model struct**

File: `bin-rag-manager/models/document/main.go`

```go
package document

import (
	"time"

	"github.com/gofrs/uuid"
)

// DocType defines the document source type
type DocType string

const (
	DocTypeUploaded  DocType = "uploaded"
	DocTypeURL       DocType = "url"
	DocTypePlatform  DocType = "platform"
	DocTypeGenerated DocType = "generated"
)

// Status defines the document processing status
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusError      Status = "error"
)

// Document represents a document within a RAG
type Document struct {
	ID             uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	CustomerID     uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	RagID          uuid.UUID  `json:"rag_id,omitempty" db:"rag_id,uuid"`
	Name           string     `json:"name,omitempty" db:"name"`
	DocType        DocType    `json:"doc_type,omitempty" db:"doc_type"`
	StorageFileID  uuid.UUID  `json:"storage_file_id,omitempty" db:"storage_file_id,uuid"`
	SourceURL      string     `json:"source_url,omitempty" db:"source_url"`
	Status         Status     `json:"status,omitempty" db:"status"`
	StatusMessage  string     `json:"status_message,omitempty" db:"status_message"`
	TMCreate       *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate       *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete       *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

**Step 2: Create field.go**

File: `bin-rag-manager/models/document/field.go`

```go
package document

// Field defines the type for document field names
type Field string

const (
	FieldID            Field = "id"
	FieldCustomerID    Field = "customer_id"
	FieldRagID         Field = "rag_id"
	FieldName          Field = "name"
	FieldDocType       Field = "doc_type"
	FieldStorageFileID Field = "storage_file_id"
	FieldSourceURL     Field = "source_url"
	FieldStatus        Field = "status"
	FieldStatusMessage Field = "status_message"
	FieldTMCreate      Field = "tm_create"
	FieldTMUpdate      Field = "tm_update"
	FieldTMDelete      Field = "tm_delete"
)
```

**Step 3: Create webhook.go**

File: `bin-rag-manager/models/document/webhook.go`

```go
package document

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Document
type WebhookMessage struct {
	ID            uuid.UUID  `json:"id,omitempty"`
	CustomerID    uuid.UUID  `json:"customer_id,omitempty"`
	RagID         uuid.UUID  `json:"rag_id,omitempty"`
	Name          string     `json:"name,omitempty"`
	DocType       DocType    `json:"doc_type,omitempty"`
	StorageFileID uuid.UUID  `json:"storage_file_id,omitempty"`
	SourceURL     string     `json:"source_url,omitempty"`
	Status        Status     `json:"status,omitempty"`
	StatusMessage string     `json:"status_message,omitempty"`
	TMCreate      *time.Time `json:"tm_create,omitempty"`
	TMUpdate      *time.Time `json:"tm_update,omitempty"`
	TMDelete      *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts the internal Document to the external representation
func (d *Document) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:            d.ID,
		CustomerID:    d.CustomerID,
		RagID:         d.RagID,
		Name:          d.Name,
		DocType:       d.DocType,
		StorageFileID: d.StorageFileID,
		SourceURL:     d.SourceURL,
		Status:        d.Status,
		StatusMessage: d.StatusMessage,
		TMCreate:      d.TMCreate,
		TMUpdate:      d.TMUpdate,
		TMDelete:      d.TMDelete,
	}
}
```

**Step 4: Verify it compiles**

```bash
cd bin-rag-manager && go build ./models/document/...
```

**Step 5: Commit**

```bash
git add bin-rag-manager/models/document/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add Document model with DocType/Status enums, Field type, and WebhookMessage"
```

---

### Task 5: Create Chunk Model

**Files:**
- Create: `bin-rag-manager/models/chunk/main.go`
- Create: `bin-rag-manager/models/chunk/field.go`

Chunks are internal-only (not exposed via API), so no webhook.go needed.

**Step 1: Create the Chunk model struct**

File: `bin-rag-manager/models/chunk/main.go`

```go
package chunk

import (
	"time"

	"github.com/gofrs/uuid"
)

// Chunk represents a text chunk with its embedding vector
type Chunk struct {
	ID           uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	DocumentID   uuid.UUID  `json:"document_id,omitempty" db:"document_id,uuid"`
	RagID        uuid.UUID  `json:"rag_id,omitempty" db:"rag_id,uuid"`
	CustomerID   uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	ChunkIndex   int        `json:"chunk_index,omitempty" db:"chunk_index"`
	Text         string     `json:"text,omitempty" db:"text"`
	SectionTitle string     `json:"section_title,omitempty" db:"section_title"`
	Embedding    []float32  `json:"-" db:"-"`
	TokenCount   int        `json:"token_count,omitempty" db:"token_count"`
	TMCreate     *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMDelete     *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}
```

Note: `Embedding` uses `db:"-"` because pgvector's `vector` type requires special handling — it cannot be scanned with standard `db:` tags. The dbhandler will handle embedding columns explicitly.

**Step 2: Create field.go**

File: `bin-rag-manager/models/chunk/field.go`

```go
package chunk

// Field defines the type for chunk field names
type Field string

const (
	FieldID           Field = "id"
	FieldDocumentID   Field = "document_id"
	FieldRagID        Field = "rag_id"
	FieldCustomerID   Field = "customer_id"
	FieldChunkIndex   Field = "chunk_index"
	FieldText         Field = "text"
	FieldSectionTitle Field = "section_title"
	FieldEmbedding    Field = "embedding"
	FieldTokenCount   Field = "token_count"
	FieldTMCreate     Field = "tm_create"
	FieldTMDelete     Field = "tm_delete"
)
```

**Step 3: Verify it compiles**

```bash
cd bin-rag-manager && go build ./models/chunk/...
```

**Step 4: Commit**

```bash
git add bin-rag-manager/models/chunk/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add Chunk model with Field type for pgvector storage"
```

---

### Task 6: Create Query Model

**Files:**
- Create: `bin-rag-manager/models/query/main.go`

The query request/response types currently live in raghandler. Move them to a proper model package so they can be shared across packages.

**Step 1: Create query model**

File: `bin-rag-manager/models/query/main.go`

```go
package query

import (
	"github.com/gofrs/uuid"
)

// Request represents a RAG query request
type Request struct {
	RagID uuid.UUID `json:"rag_id"`
	Query string    `json:"query"`
	TopK  int       `json:"top_k,omitempty"`
}

// Source represents a source reference in the query response
type Source struct {
	DocumentID    uuid.UUID `json:"document_id"`
	DocumentName  string    `json:"document_name"`
	SectionTitle  string    `json:"section_title"`
	RelevanceScore float64  `json:"relevance_score"`
}

// Response represents a RAG query response
type Response struct {
	Answer  string   `json:"answer"`
	Sources []Source `json:"sources"`
}
```

**Step 2: Verify it compiles**

```bash
cd bin-rag-manager && go build ./models/query/...
```

**Step 3: Commit**

```bash
git add bin-rag-manager/models/query/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add Query request/response models with rag_id scoping"
```

---

### Task 7: Create PostgreSQL DBHandler Interface

**Files:**
- Create: `bin-rag-manager/pkg/dbhandler/main.go`

**Step 1: Create the DBHandler interface**

File: `bin-rag-manager/pkg/dbhandler/main.go`

```go
package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/chunk"
	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/rag"
)

// DBHandler defines all database operations for rag-manager
type DBHandler interface {
	// Rag operations
	RagCreate(ctx context.Context, r *rag.Rag) error
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error)
	RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) error
	RagDelete(ctx context.Context, id uuid.UUID) error

	// Document operations
	DocumentCreate(ctx context.Context, d *document.Document) error
	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error)
	DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error)
	DocumentUpdate(ctx context.Context, id uuid.UUID, fields map[document.Field]any) error
	DocumentDelete(ctx context.Context, id uuid.UUID) error
	DocumentDeleteByRagID(ctx context.Context, ragID uuid.UUID) error

	// Chunk operations
	ChunkCreate(ctx context.Context, c *chunk.Chunk, embedding []float32) error
	ChunkCreateBatch(ctx context.Context, chunks []*chunk.Chunk, embeddings [][]float32) error
	ChunkSearchByRagID(ctx context.Context, ragID uuid.UUID, queryEmbedding []float32, topK int) ([]*chunk.Chunk, []float64, error)
	ChunkDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
	ChunkDeleteByRagID(ctx context.Context, ragID uuid.UUID) error
	ChunkSoftDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
	ChunkSoftDeleteByRagID(ctx context.Context, ragID uuid.UUID) error
}

// handler implements DBHandler using PostgreSQL
type handler struct {
	db *sql.DB
}

// NewHandler creates a new DBHandler
func NewHandler(db *sql.DB) DBHandler {
	return &handler{
		db: db,
	}
}
```

**Step 2: Generate mock**

```bash
cd bin-rag-manager && go generate ./pkg/dbhandler/...
```

**Step 3: Verify it compiles**

```bash
cd bin-rag-manager && go build ./pkg/dbhandler/...
```

**Step 4: Commit**

```bash
git add bin-rag-manager/pkg/dbhandler/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Add DBHandler interface for PostgreSQL operations (rag, document, chunk CRUD)"
```

---

### Task 8: Implement DBHandler — Rag Operations

**Files:**
- Create: `bin-rag-manager/pkg/dbhandler/rag.go`
- Create: `bin-rag-manager/pkg/dbhandler/rag_test.go`

**Step 1: Implement rag CRUD operations**

File: `bin-rag-manager/pkg/dbhandler/rag.go`

Implement `RagCreate`, `RagGet`, `RagGetsByCustomerID`, `RagUpdate`, `RagDelete` using squirrel query builder on table `rag_rags`. Follow these patterns:

- Use `sq.Insert("rag_rags")`, `sq.Select(...)`, `sq.Update("rag_rags")`
- Use `sq.Dollar` placeholder format (PostgreSQL uses `$1` not `?`)
- UUID fields: use `.Bytes()` for insert/where, `uuid.FromBytes()` for scanning
- Soft delete: `RagDelete` sets `tm_delete = NOW()` and `tm_update = NOW()`
- List queries: add `WHERE tm_delete IS NULL`
- Initialize result slices as `res := []*rag.Rag{}` (never nil)

**Step 2: Write tests**

Test all CRUD operations with a mock or test database. At minimum, verify the SQL queries are built correctly by checking the squirrel output with `.ToSql()`.

**Step 3: Run tests**

```bash
cd bin-rag-manager && go test -v ./pkg/dbhandler/...
```

**Step 4: Commit**

```bash
git add bin-rag-manager/pkg/dbhandler/rag.go bin-rag-manager/pkg/dbhandler/rag_test.go
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Implement Rag CRUD operations in dbhandler with squirrel"
```

---

### Task 9: Implement DBHandler — Document Operations

**Files:**
- Create: `bin-rag-manager/pkg/dbhandler/document.go`
- Create: `bin-rag-manager/pkg/dbhandler/document_test.go`

**Step 1: Implement document CRUD operations**

File: `bin-rag-manager/pkg/dbhandler/document.go`

Implement all Document methods from the DBHandler interface. Same patterns as rag operations. Additional notes:

- `DocumentGetsByRagID`: filter by `rag_id` AND `tm_delete IS NULL`
- `DocumentDeleteByRagID`: soft-delete all documents for a RAG (used when deleting a RAG)
- `DocumentUpdate`: used to update status, status_message during async processing

**Step 2: Write tests and run**

```bash
cd bin-rag-manager && go test -v ./pkg/dbhandler/...
```

**Step 3: Commit**

```bash
git add bin-rag-manager/pkg/dbhandler/document.go bin-rag-manager/pkg/dbhandler/document_test.go
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Implement Document CRUD operations in dbhandler"
```

---

### Task 10: Implement DBHandler — Chunk Operations

**Files:**
- Create: `bin-rag-manager/pkg/dbhandler/chunk.go`
- Create: `bin-rag-manager/pkg/dbhandler/chunk_test.go`

**Step 1: Implement chunk operations**

File: `bin-rag-manager/pkg/dbhandler/chunk.go`

This is the most complex task because of pgvector handling:

- `ChunkCreate`: INSERT with embedding as `$N::vector` cast
- `ChunkCreateBatch`: batch insert for efficiency during document processing
- `ChunkSearchByRagID`: the core vector search query:
  ```sql
  SELECT id, document_id, rag_id, customer_id, chunk_index, text, section_title, token_count, tm_create,
         embedding <=> $1::vector AS distance
  FROM rag_chunks
  WHERE rag_id = $2 AND tm_delete IS NULL
  ORDER BY embedding <=> $1::vector
  LIMIT $3
  ```
  Returns chunks and their distance scores. Convert distance to relevance: `relevance = 1 - distance`.
- `ChunkSoftDeleteByDocumentID/RagID`: set `tm_delete = NOW()`
- `ChunkDeleteByDocumentID/RagID`: hard delete (for re-processing a document)

**Important**: Embedding vectors must be formatted as pgvector string format: `[0.1,0.2,0.3,...]` for insertion.

**Step 2: Write tests and run**

```bash
cd bin-rag-manager && go test -v ./pkg/dbhandler/...
```

**Step 3: Commit**

```bash
git add bin-rag-manager/pkg/dbhandler/chunk.go bin-rag-manager/pkg/dbhandler/chunk_test.go
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Implement Chunk operations with pgvector search in dbhandler"
```

---

### Task 11: Create Alembic Migration for PostgreSQL

**Files:**
- Create: `bin-dbscheme-manager/rag-manager/alembic.ini`
- Create: `bin-dbscheme-manager/rag-manager/env.py`
- Create: `bin-dbscheme-manager/rag-manager/versions/001_create_rag_tables.py`

**Step 1: Create the Alembic environment for PostgreSQL**

Set up `bin-dbscheme-manager/rag-manager/` as a new Alembic environment targeting PostgreSQL. Model it after `bin-dbscheme-manager/bin-manager/` but with PostgreSQL connection string.

**Step 2: Create the migration file**

File: `bin-dbscheme-manager/rag-manager/versions/001_create_rag_tables.py`

```python
def upgrade():
    op.execute("CREATE EXTENSION IF NOT EXISTS vector")

    op.execute("""
    CREATE TABLE rag_rags (
        id UUID PRIMARY KEY,
        customer_id UUID NOT NULL,
        name TEXT NOT NULL,
        description TEXT DEFAULT '',
        tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_delete TIMESTAMP WITH TIME ZONE
    )
    """)

    op.execute("""
    CREATE TABLE rag_documents (
        id UUID PRIMARY KEY,
        customer_id UUID NOT NULL,
        rag_id UUID NOT NULL REFERENCES rag_rags(id),
        name TEXT NOT NULL,
        doc_type TEXT NOT NULL DEFAULT 'uploaded',
        storage_file_id UUID,
        source_url TEXT,
        status TEXT NOT NULL DEFAULT 'pending',
        status_message TEXT,
        tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_update TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_delete TIMESTAMP WITH TIME ZONE
    )
    """)

    op.execute("""
    CREATE TABLE rag_chunks (
        id UUID PRIMARY KEY,
        document_id UUID NOT NULL REFERENCES rag_documents(id),
        rag_id UUID NOT NULL,
        customer_id UUID NOT NULL,
        chunk_index INTEGER NOT NULL DEFAULT 0,
        text TEXT NOT NULL,
        section_title TEXT DEFAULT '',
        embedding vector(1536),
        token_count INTEGER DEFAULT 0,
        tm_create TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        tm_delete TIMESTAMP WITH TIME ZONE
    )
    """)

    # Indexes
    op.execute("CREATE INDEX idx_rag_rags_customer_id ON rag_rags(customer_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_documents_rag_id ON rag_documents(rag_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_documents_customer_id ON rag_documents(customer_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_chunks_rag_id ON rag_chunks(rag_id) WHERE tm_delete IS NULL")
    op.execute("CREATE INDEX idx_rag_chunks_document_id ON rag_chunks(document_id) WHERE tm_delete IS NULL")

    # HNSW index for vector search
    op.execute("""
    CREATE INDEX idx_rag_chunks_embedding ON rag_chunks
    USING hnsw (embedding vector_cosine_ops)
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS rag_chunks")
    op.execute("DROP TABLE IF EXISTS rag_documents")
    op.execute("DROP TABLE IF EXISTS rag_rags")
    op.execute("DROP EXTENSION IF EXISTS vector")
```

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/rag-manager/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-dbscheme-manager: Add Alembic migration environment for PostgreSQL RAG tables
- bin-dbscheme-manager: Create rag_rags, rag_documents, rag_chunks tables with pgvector HNSW index"
```

**IMPORTANT: Do NOT run `alembic upgrade`. Only create the migration files.**

---

### Task 12: Wire PostgreSQL Into Service Startup

**Files:**
- Modify: `bin-rag-manager/cmd/rag-manager/main.go`

**Step 1: Add database connection to runService()**

In `runService()`, after the RabbitMQ connection setup, add PostgreSQL connection:

```go
import (
	"database/sql"
	_ "github.com/lib/pq"
	"monorepo/bin-rag-manager/pkg/dbhandler"
)

// In runService():
db, err := sql.Open("postgres", cfg.PostgreSQLDSN)
if err != nil {
	return fmt.Errorf("could not connect to PostgreSQL: %w", err)
}
defer db.Close()

if err := db.Ping(); err != nil {
	return fmt.Errorf("could not ping PostgreSQL: %w", err)
}
log.Info("Connected to PostgreSQL")

dbh := dbhandler.NewHandler(db)
```

**Step 2: Pass dbhandler to raghandler**

Update the `raghandler.NewRagHandler()` call to accept the `dbhandler.DBHandler` as a new parameter. This will require updating the raghandler interface in the next task.

For now, add `dbh` as a parameter and update the call:

```go
ragH := raghandler.NewRagHandler(ret, gen, emb, vectorStore, dbh, cfg.RAGDocsBasePath, cfg.GCSEmbeddingsPath, cfg.RAGTopK)
```

**Step 3: Update K8s deployment**

Add `POSTGRESQL_DSN` env var to `bin-rag-manager/k8s/deployment.yml`.

**Step 4: Verify it compiles**

```bash
cd bin-rag-manager && go build ./cmd/...
```

**Step 5: Commit**

```bash
git add bin-rag-manager/cmd/rag-manager/main.go bin-rag-manager/k8s/deployment.yml
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Wire PostgreSQL connection and dbhandler into service startup
- bin-rag-manager: Add POSTGRESQL_DSN to K8s deployment"
```

---

### Task 13: Refactor RagHandler to Accept DBHandler

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/main.go`

**Step 1: Add dbhandler dependency to ragHandler struct**

Add `dbHandler dbhandler.DBHandler` to the `ragHandler` struct and update `NewRagHandler` to accept it. Update the `RagHandler` interface to add new CRUD methods:

```go
type RagHandler interface {
	// Existing (kept for backward compat during migration)
	Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
	IndexFull(ctx context.Context) error
	IndexIncremental(ctx context.Context, files []string) error
	IndexStatus(ctx context.Context) (*IndexStatusResponse, error)

	// New multi-tenant operations
	RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, fileIDs []uuid.UUID, urls []string) (*rag.Rag, error)
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error)
	RagDelete(ctx context.Context, id uuid.UUID) error

	DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, fileIDs []uuid.UUID, urls []string) ([]*document.Document, error)
	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error)
	DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error)
	DocumentDelete(ctx context.Context, id uuid.UUID) error

	QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error)
}
```

**Step 2: Stub the new methods**

Add stub implementations that return `nil, fmt.Errorf("not implemented")` for each new method. They will be implemented in later tasks.

**Step 3: Regenerate mocks**

```bash
cd bin-rag-manager && go generate ./pkg/raghandler/...
```

**Step 4: Run tests**

```bash
cd bin-rag-manager && go test ./...
```

**Step 5: Commit**

```bash
git add bin-rag-manager/pkg/raghandler/
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Extend RagHandler interface with multi-tenant CRUD and query methods
- bin-rag-manager: Add dbhandler dependency to ragHandler struct"
```

---

### Task 14: Run Full Verification

**Step 1: Run the full verification workflow**

```bash
cd bin-rag-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Fix any issues found**

Address lint warnings, unused imports, test failures.

**Step 3: Final commit**

```bash
git add -A
git commit -m "NOJIRA-Rag-manager-postgresql-foundation

- bin-rag-manager: Fix lint and test issues from full verification"
```

---

## Summary of What Phase 1 Delivers

After completing all 14 tasks:

1. **Models**: `rag.Rag`, `document.Document`, `chunk.Chunk`, `query.Request/Response` — proper Go structs with db tags, Field types, WebhookMessage patterns
2. **DBHandler**: PostgreSQL-specific database layer with CRUD for all three tables + pgvector similarity search
3. **Alembic migration**: Schema for `rag_rags`, `rag_documents`, `rag_chunks` with HNSW index
4. **Config**: `POSTGRESQL_DSN` environment variable wired into service startup
5. **RagHandler**: Extended interface with stubs for new multi-tenant operations
6. **Service startup**: PostgreSQL connection established, dbhandler passed to raghandler

## What Comes Next (Phase 2)

- Implement the stubbed RagHandler methods (CRUD + QueryRag)
- Add new ListenHandler routes (`/v1/rags`, `/v1/documents`, `/v1/query`)
- Add bin-common-handler RPC methods
- OpenAPI schemas + API manager integration
- AI talk integration (use_rag + rag_id)
