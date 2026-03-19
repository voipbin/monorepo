# RAG Document Ingestion Pipeline Design

## Date
2026-03-19

## Problem Statement
rag-manager accepts document creation requests with a `storage_file_id` reference or a `source_url` but never fetches or processes the file content. Documents remain permanently in `pending` status with no chunking, embedding, or indexing. The RAG query pipeline requires indexed chunks to function.

Additionally, the current API requires a 3-step flow (create RAG → upload file → create document) which leaks internal implementation details. Users shouldn't need to know about documents — they just want to create a knowledge base with sources.

## Goal
Implement an async ingestion pipeline that:
1. Fetches file content — either from GCS (via storage-manager metadata) or from a URL
2. Downloads the file to a temp file
3. Chunks the content using format-appropriate parsers
4. Generates embeddings via Google Gemini
5. Stores chunks with embeddings in PostgreSQL (pgvector)
6. Updates document status to `ready` or `error`

Simplify the API so users create a RAG with sources in a single request.

Include safeguards for stuck documents (heartbeat, startup sweep, periodic ticker) and bounded retries.

## Simplified API Flow

### Current Flow (before this change)
1. `POST /rags` — create empty RAG
2. `POST /storage/files` — upload file (separate service)
3. `POST /rag-documents` — create document linking file to RAG
4. User must track document status separately

### New Flow (after this change)
1. User uploads files to storage-manager anytime (existing, unchanged)
2. `POST /rags` — create RAG with file IDs and/or URLs in one request
3. rag-manager internally creates documents and triggers ingestion
4. `GET /rags/{id}` — returns RAG with status and per-source progress
5. `POST /rags/{id}/sources` — add more sources to an existing RAG

Documents are an internal implementation detail. Users see "sources" in the RAG response.

### RAG Create Request (new)
```json
POST /rags
{
  "name": "My Knowledge Base",
  "description": "Product docs",
  "storage_file_ids": ["uuid1", "uuid2"],
  "source_urls": ["https://example.com/doc.pdf"]
}
```

At least one `storage_file_ids` or `source_urls` entry is required. Both can be provided.

### Add Sources Request (new endpoint)
```json
POST /rags/{id}/sources
{
  "storage_file_ids": ["uuid3"],
  "source_urls": ["https://example.com/api.yaml"]
}
```

Returns the updated RAG (same format as `GET /rags/{id}`).

### RAG Response (updated)
```json
{
  "id": "rag-uuid",
  "name": "My Knowledge Base",
  "description": "Product docs",
  "status": "processing",
  "sources": [
    {"storage_file_id": "uuid1", "status": "ready"},
    {"storage_file_id": "uuid2", "status": "processing"},
    {"source_url": "https://example.com/doc.pdf", "status": "pending"}
  ],
  "tm_create": "2026-03-19T10:00:00Z",
  "tm_update": "2026-03-19T10:01:00Z"
}
```

### RAG Status Derivation
RAG status is computed from its documents at read time (not stored):
- `pending` — all documents are pending
- `processing` — at least one document is pending or processing
- `ready` — all documents are ready
- `error` — no documents are pending/processing, and at least one is in error

### Document Endpoints
Keep document endpoints (`GET /rag-documents`, `GET /rag-documents/{id}`) as **read-only** for debugging per-source status. Remove `POST /rag-documents` and `DELETE /rag-documents/{id}` from the public API — documents are managed internally via RAG creation and source addition.

## Document Status Flow

```
pending --> processing --> ready
  ^             |
  |             v
  +-------- (failed, retry_count < 3)
                |
                v
             error (retry_count >= 3, terminal)
```

- `pending`: document created, awaiting ingestion
- `processing`: actively being ingested, heartbeat (`tm_processing`) updated periodically
- `ready`: ingestion complete, chunks indexed and queryable
- `error`: ingestion failed after max retries (terminal state)

## RAG Model — Transient Fields

The `Rag` struct gets `Status` and `Sources` as **transient fields** (no `db:` tag). These are:
- Populated by raghandler after fetching from DB (enriched from associated documents)
- Serialized over RabbitMQ RPC via JSON (no `db:` tag means DB operations ignore them)
- Copied through by `ConvertWebhookMessage()` naturally

```go
type Rag struct {
    // DB-backed fields
    ID          uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
    CustomerID  uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
    Name        string     `json:"name,omitempty" db:"name"`
    Description string     `json:"description,omitempty" db:"description"`
    TMCreate    *time.Time `json:"tm_create,omitempty" db:"tm_create"`
    TMUpdate    *time.Time `json:"tm_update,omitempty" db:"tm_update"`
    TMDelete    *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`

    // Transient — populated by handler, ignored by DB (no db tag)
    Status  rmdocument.Status `json:"status,omitempty"`
    Sources []Source          `json:"sources,omitempty"`
}
```

This approach keeps:
- raghandler returning `*rag.Rag` (consistent type, no WebhookMessage leaking into business logic)
- DB operations unaffected (PrepareFields/ScanRow only process fields with `db:` tags)
- RPC transport preserving Status/Sources via JSON serialization
- api-manager's `ragGet()` + `.ConvertWebhookMessage()` flow unchanged
- No dual source of truth, no denormalization

### Source Struct

```go
type Source struct {
    StorageFileID *uuid.UUID      `json:"storage_file_id,omitempty"`
    SourceURL     string          `json:"source_url,omitempty"`
    Status        document.Status `json:"status,omitempty"`
    StatusMessage string          `json:"status_message,omitempty"`
}
```

`StorageFileID` is `*uuid.UUID` (pointer) so nil UUIDs are properly omitted from JSON with `omitempty`.

## Document Model Changes

Add two fields to the `Document` struct and `rag_documents` table:

| Field | Go Type | DB Column | Description |
|-------|---------|-----------|-------------|
| `RetryCount` | `int` | `retry_count INTEGER NOT NULL DEFAULT 0` | Number of ingestion attempts |
| `TMProcessing` | `*time.Time` | `tm_processing TIMESTAMP WITH TIME ZONE` | Heartbeat timestamp, updated during active processing |

Requires a new golang-migrate migration file: `000002_add_document_ingestion_fields.up.sql` / `.down.sql`.

## New Dependencies

### Service Dependencies
- **`requesthandler`** (bin-common-handler) - to call `StorageV1FileGet()` for file metadata (bucket name, filepath, filename)
- **GCS client** (`cloud.google.com/go/storage`) - to download files directly from the bucket

### File Parsing Libraries
| Format | Extension(s) | Library | Notes |
|--------|-------------|---------|-------|
| RST | `.rst` | existing chunker | Already implemented |
| Markdown | `.md` | existing chunker | Already implemented |
| OpenAPI YAML | `.yaml`, `.yml` | existing chunker | Already implemented |
| Plain text | `.txt`, fallback | stdlib (`io`) | New, simple split by token count |
| PDF | `.pdf` | `github.com/ledongthuc/pdf` | New |
| HTML | `.html`, `.htm` | `golang.org/x/net/html` | New |
| CSV | `.csv` | `encoding/csv` (stdlib) | New |
| DOCX | `.docx` | `github.com/fumiama/go-docx` | New |
| JSON | `.json` | `encoding/json` (stdlib) | New |

Format detection: file extension from the storage-manager filename or URL path, then HTTP Content-Type header (for URL sources), fallback to plain text.

## Ingestion Flow

### Trigger
`RagCreate` creates the RAG, then for each file ID / URL creates a document with `status=pending` and launches `go h.documentIngest(doc)` using `context.Background()` (not the request context, which gets cancelled after the RPC response).

Similarly, `RagAddSources` creates documents for the new sources and triggers ingestion.

### Pipeline Steps (`documentIngest`)

1. **Atomic claim**: `UPDATE rag_documents SET status='processing', tm_processing=NOW(), retry_count=retry_count+1 WHERE id=? AND status='pending' RETURNING *`. If no row returned, another pod claimed it — abort silently.

2. **Acquire file**: Depending on document source:
   - **`storage_file_id` set (uploaded files)**: Call `reqHandler.StorageV1FileGet(ctx, storageFileID)` to get `BucketName`, `Filepath`, `Filename`, and `Filesize`. Check file size against 50 MB limit. Download from GCS bucket to temp file.
   - **`source_url` set (URL-sourced files)**: Download the URL to a temp file via HTTP GET. Enforce 50 MB limit by checking `Content-Length` header before download and capping the reader during download. Determine filename from URL path.

3. **Detect format** (priority order):
   1. File extension from filename (storage-manager `Filename` or URL path)
   2. HTTP `Content-Type` header (for URL sources only, e.g., `application/pdf` → `.pdf`)
   3. Fallback to plain text

4. **Chunk content**: Run the appropriate chunker with the temp file path and max 512 tokens per chunk.

5. **Update heartbeat**: Set `tm_processing=NOW()` after chunking completes.

6. **Generate embeddings**: Call `EmbedTexts()` sequentially for all chunks. Update `tm_processing` every 10 chunks as a heartbeat.

7. **Store chunks**: Batch-insert chunks with embeddings into `rag_chunks` table.

8. **Set status**: `status=ready`.

9. **On error**:
    - If `retry_count >= 3`: set `status=error`, `status_message=<error detail>` (terminal)
    - If `retry_count < 3`: set `status=pending` (will be re-picked by ticker)

## Stuck Detection & Recovery

### 1. Startup Sweep
On boot, before starting the listen handler:
- Find all documents where `status=processing` (stale from previous pod lifecycle) — reset to `status=pending`
- Find all documents where `status=pending` AND `retry_count < 3` — trigger `go h.documentIngest(doc)` for each

### 2. Periodic Ticker
Every 5 minutes:
- Find documents where `status=processing` AND `tm_processing` older than 5 minutes — reset to `status=pending`
- Find documents where `status=pending` AND `retry_count < 3` — trigger `go h.documentIngest(doc)` for each

Both mechanisms use the atomic claim (`UPDATE ... WHERE status='pending' RETURNING *`) so multiple pods never process the same document concurrently.

## Concurrency Safety
- **Atomic claiming** prevents duplicate processing across pods. Only the pod that successfully transitions `status` from `pending` to `processing` proceeds.
- **Heartbeat** (`tm_processing`) distinguishes actively-processing documents from stuck ones.
- **Bounded retries** (max 3) prevent infinite loops on permanently broken documents.

## Max File Size
50 MB limit for ingestion. For GCS files, checked via storage-manager file metadata before downloading. For URL sources, checked via `Content-Length` header (when available) and enforced by capping the HTTP response reader with `io.LimitReader` during download. Files exceeding the limit are immediately marked `status=error` with `status_message` explaining the size limit.

## File Handling
Files are downloaded to a local temp file (`os.CreateTemp`), then the file path is passed to chunkers. Two source paths:

### GCS Files (`storage_file_id` set)
- Call `StorageV1FileGet` for metadata (bucket, filepath, filename, filesize)
- Download from GCS bucket using `BucketReader.DownloadToTempFile()`
- `defer os.Remove()` ensures cleanup

### URL Files (`source_url` set)
- HTTP GET the URL with a capped reader (`io.LimitReader` at 50 MB)
- Save response body to temp file
- Detect format: file extension from URL path first, then `Content-Type` header mapping, fallback to plain text
- `defer os.Remove()` ensures cleanup

### Content-Type to Extension Mapping (URL sources)
| Content-Type | Extension |
|-------------|-----------|
| `text/plain` | `.txt` |
| `text/html` | `.html` |
| `text/csv` | `.csv` |
| `text/markdown` | `.md` |
| `application/pdf` | `.pdf` |
| `application/json` | `.json` |
| `application/vnd.openxmlformats-officedocument.wordprocessingml.document` | `.docx` |
| `text/x-rst`, `text/restructuredtext` | `.rst` |
| `application/x-yaml`, `text/yaml` | `.yaml` |

Both paths:
- Align with the existing chunker interface: `Chunk(filePath string, maxTokens int)`
- Keep memory usage low (no full file in memory)
- Temp file is cleaned up immediately after processing

## Wiring Changes

### `cmd/rag-manager/main.go`
1. Add `requesthandler` — create via `requesthandler.NewRequestHandler(sockHandler, ...)`
2. Add GCS `storage.Client` — create via `storage.NewClient(ctx)` (uses Workload Identity)
3. Add config flag `gcp_bucket_name_media` for the media bucket name
4. Update `raghandler.NewRagHandler(emb, dbH, reqHandler, gcsClient, bucketName)`
5. After creating `ragH`, run startup sweep: `ragH.DocumentIngestPendingAll(ctx)`
6. Start periodic ticker goroutine: `go ragH.RunIngestionTicker(ctx, 5*time.Minute)`

### `ragHandler` struct
```go
type ragHandler struct {
    embedder     embedder.Embedder
    dbHandler    dbhandler.DBHandler
    reqHandler   requesthandler.RequestHandler
    gcsClient    *storage.Client
    bucketName   string
}
```

### Updated `RagHandler` Interface Changes

#### Modified Methods
- `RagCreate(ctx, customerID, name, description, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error)` — now creates RAG + documents + triggers ingestion, returns enriched Rag with Status/Sources
- `RagGet(ctx, id) (*rag.Rag, error)` — returns Rag with computed Status and Sources populated (transient fields)
- `RagList(ctx, size, token, filters) ([]*rag.Rag, error)` — returns Rags with Status/Sources populated via batch document fetch

#### New Methods
- `RagAddSources(ctx, ragID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error)` — adds sources to existing RAG, creates documents, triggers ingestion, returns enriched Rag
- `documentIngest(doc *document.Document)` — core ingestion pipeline (private)
- `documentAcquireFile(ctx, doc) (tmpPath, filename, contentType string, err error)` — dispatches to GCS or URL download (private)
- `documentDownloadURL(ctx, sourceURL string) (tmpPath, filename, contentType string, err error)` — downloads URL to temp file (private)
- `DocumentIngestPendingAll(ctx)` — startup sweep
- `RunIngestionTicker(ctx, interval)` — periodic ticker loop

### New `DBHandler` Methods
- `DocumentClaimForProcessing(ctx, docID) (*document.Document, error)` — atomic `UPDATE ... SET status='processing' WHERE status='pending' RETURNING *`
- `DocumentUpdateHeartbeat(ctx, docID) error` — updates `tm_processing=NOW()`
- `DocumentGetStale(ctx, threshold) ([]*document.Document, error)` — finds `processing` docs with heartbeat older than threshold
- `DocumentGetPending(ctx) ([]*document.Document, error)` — finds `pending` docs with `retry_count < 3`
- `DocumentResetStaleToPending(ctx, threshold) error` — resets stale `processing` docs back to `pending`
- `DocumentGetsByRagID(ctx, ragID) ([]*document.Document, error)` — get all active documents for a RAG
- `DocumentGetsByRagIDs(ctx, ragIDs []uuid.UUID) (map[uuid.UUID][]*document.Document, error)` — batch fetch for RagList (single query with `IN` clause)

### RAG Response Building (Transient Field Enrichment)
When returning a RAG (GET or after create/add-sources), raghandler enriches the `*rag.Rag` directly:
1. Fetch the RAG from DB → `*rag.Rag` (Status and Sources are zero-valued)
2. Fetch all documents for this RAG (not soft-deleted)
3. Compute RAG status from documents → set `r.Status`
4. Build sources list from documents → set `r.Sources`
5. Return the enriched `*rag.Rag`

For `RagList`, use batch `DocumentGetsByRagIDs` (single `SELECT ... WHERE rag_id IN (...)` query) to avoid N+1.

The enriched `*rag.Rag` travels over RabbitMQ RPC (JSON includes transient fields). On the api-manager side, `ragGet()` returns the enriched `*rmrag.Rag`, and `.ConvertWebhookMessage()` copies Status/Sources to the external representation. No type conflicts, no extra RPC calls.

## Chunking Configuration
- Max tokens per chunk: 512
- Chunker interface (existing): `Chunk(filePath string, maxTokens int) ([]Chunk, error)`
- New chunkers implement the same interface

## Embedding Configuration
- Model: text-embedding-004 (768 dimensions, existing)
- Sequential processing (no concurrency)
- Heartbeat updated every 10 chunks during embedding

## Files to Create/Modify

### New Files
- `bin-rag-manager/migrations/000002_add_document_ingestion_fields.up.sql`
- `bin-rag-manager/migrations/000002_add_document_ingestion_fields.down.sql`
- `bin-rag-manager/pkg/chunker/text.go` — plain text chunker
- `bin-rag-manager/pkg/chunker/pdf.go` — PDF chunker
- `bin-rag-manager/pkg/chunker/html.go` — HTML chunker
- `bin-rag-manager/pkg/chunker/csv.go` — CSV chunker
- `bin-rag-manager/pkg/chunker/docx.go` — DOCX chunker
- `bin-rag-manager/pkg/chunker/json.go` — JSON chunker
- `bin-rag-manager/pkg/chunker/selector.go` — format dispatcher by extension/content-type
- `bin-rag-manager/pkg/bucketreader/main.go` — GCS BucketReader interface + implementation

### Modified Files — rag-manager
- `bin-rag-manager/models/document/main.go` — add `RetryCount`, `TMProcessing` fields
- `bin-rag-manager/models/document/field.go` — add new field constants
- `bin-rag-manager/models/rag/main.go` — add `Status` and `Sources` as transient fields (no `db:` tag)
- `bin-rag-manager/models/rag/webhook.go` — add `Status` and `Sources` to WebhookMessage, update `ConvertWebhookMessage()`
- `bin-rag-manager/pkg/dbhandler/main.go` — add new DBHandler interface methods (including batch `DocumentGetsByRagIDs`)
- `bin-rag-manager/pkg/dbhandler/document.go` — implement new DB methods
- `bin-rag-manager/pkg/raghandler/main.go` — update struct, constructor, interface (add RagAddSources)
- `bin-rag-manager/pkg/raghandler/rag.go` — update RagCreate to accept file IDs/URLs, add RagAddSources, update RagGet to compute status
- `bin-rag-manager/pkg/raghandler/document.go` — add ingestion pipeline, heartbeat, sweep, ticker
- `bin-rag-manager/pkg/listenhandler/v1_rags.go` — update create handler for new params, add sources endpoint
- `bin-rag-manager/pkg/listenhandler/v1_documents.go` — remove POST handler, keep GET only
- `bin-rag-manager/cmd/rag-manager/main.go` — wire requesthandler, GCS client, startup sweep, ticker
- `bin-rag-manager/internal/config/config.go` — add `gcp_bucket_name_media` config

### Modified Files — api-manager
- `bin-api-manager/server/rags.go` — update PostRags to pass file IDs/URLs, add PostRagsIdSources handler
- `bin-api-manager/server/rag_documents.go` — remove PostRagDocuments and DeleteRagDocumentsId handlers
- `bin-api-manager/pkg/servicehandler/rag.go` — update RagCreate signature, add RagAddSources
- `bin-api-manager/pkg/servicehandler/rag_document.go` — remove RagDocumentCreate and RagDocumentDelete

### Modified Files — common-handler
- `bin-common-handler/pkg/requesthandler/rag_rags.go` — update RagV1RagCreate to accept file IDs/URLs, add RagV1RagAddSources
- `bin-common-handler/pkg/requesthandler/rag_documents.go` — remove RagV1DocumentCreate and RagV1DocumentDelete

### Modified Files — openapi-manager
- `bin-openapi-manager/openapi/openapi.yaml` — update RAG create request schema (add storage_file_ids, source_urls), update RAG response schema (add status, sources), add POST /rags/{id}/sources endpoint, remove POST /rag-documents and DELETE /rag-documents/{id}

## DocType Change

**Remove `doc_type` from the API entirely.** rag-manager auto-derives it internally when creating documents:
- `storage_file_id` present → `DocTypeUploaded`
- `source_url` present → `DocTypeURL`

The field remains in the model and database as internal metadata. Since documents are now created internally (not by users), there's no risk of mismatched metadata.

## Trade-offs
- **Sequential embedding over concurrent**: Simpler, avoids rate limiting. Can optimize later if ingestion latency becomes a concern.
- **Temp file over streaming**: Some parsers (PDF, DOCX) require random access (`io.ReaderAt`). Temp file is the simplest approach that works for all formats.
- **50 MB limit**: Arbitrary but reasonable. Protects disk and memory. Can be made configurable later.
- **5-minute heartbeat threshold**: Balances between catching truly stuck documents and allowing legitimate slow processing. Large documents with many chunks may take several minutes.
- **Transient fields on Rag struct**: Status and Sources are computed at read time and set as transient fields (no `db:` tag). This avoids a stored status column, keeps the DB schema clean, and lets the enriched Rag travel naturally over RPC via JSON. The only cost is an extra documents query per RAG read (batch query for list).
- **Sources in RAG response**: Slightly violates atomic response principle, but sources are same-service data and documents are hidden from public API. Users need source-level visibility without a separate endpoint.
