# RAG Manager Multi-Tenant Database Design

## Problem Statement

The current bin-rag-manager is an internal-only service that indexes VoIPbin documentation into an in-memory vector store with gob file persistence. This architecture has several limitations:

- **Single-tenant**: No customer isolation — all data in one store
- **No persistence guarantees**: In-memory store with optional file backup; data lost on pod restart without recent indexing
- **Not scalable**: Single replica only, all embeddings held in RAM
- **Internal-only**: No customer-facing API for managing knowledge bases

The goal is to transform bin-rag-manager into a multi-tenant, database-backed RAG system where customers can create their own knowledge bases, upload documents, and query them — both via REST API and through AI talk actions.

## Data Model

### Database: PostgreSQL 17 (Cloud SQL) + pgvector extension

PostgreSQL client is **isolated to bin-rag-manager only**. Other services continue using MySQL. The pgvector extension is enabled with `CREATE EXTENSION vector;` on Cloud SQL.

### Tables

#### rag_rags (KB container)

A customer can have multiple RAGs (e.g., one for support docs, one for product specs).

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| customer_id | UUID | Owner |
| name | text | Display name |
| description | text | Optional description |
| tm_create | datetime | Created timestamp |
| tm_update | datetime | Last updated |
| tm_delete | datetime | Soft delete |

#### rag_documents (belongs to one RAG)

Each document belongs to exactly one RAG. If a customer wants the same document in two RAGs, they upload it separately. Raw files are stored in GCS via bin-storage-manager. URL-sourced documents are fetched, stored in GCS, and tracked with their original URL.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| customer_id | UUID | Owner |
| rag_id | UUID | FK → rag_rags |
| name | text | Display name |
| doc_type | enum | uploaded, url, platform, generated |
| storage_file_id | UUID | Reference to bin-storage-manager resource |
| source_url | text | Original URL (for doc_type=url, null otherwise) |
| status | enum | pending, processing, ready, error |
| status_message | text | Error detail when status=error, null otherwise |
| tm_create | datetime | Created timestamp |
| tm_update | datetime | Last updated |
| tm_delete | datetime | Soft delete |

#### rag_chunks (denormalized rag_id for fast vector search)

Chunks are the atomic units for vector search. `rag_id` is denormalized from `rag_documents` to avoid joins during vector search — queries use a simple `WHERE rag_id = ?`.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | PK |
| document_id | UUID | FK → rag_documents |
| rag_id | UUID | Denormalized for fast search |
| customer_id | UUID | Denormalized for filtering |
| chunk_index | int | Order within document |
| text | text | Chunk content |
| section_title | text | Section header for context |
| embedding | vector(1536) | OpenAI text-embedding-3-small |
| token_count | int | Approximate token count |
| tm_create | datetime | Created timestamp |
| tm_delete | datetime | Soft delete (set when parent doc/RAG is deleted) |

**Index**: HNSW index on `embedding` column for approximate nearest neighbor search.

**Soft delete on chunks**: When a document or RAG is soft-deleted, its chunks are also soft-deleted by setting `tm_delete`. This avoids joins with `rag_documents` during vector search — the query simply adds `AND tm_delete IS NULL`.

## API Endpoints

All three resource types are independent top-level resources. The old internal-only `POST /v1/rags/query` endpoint is retired.

### RAG Management

```
POST   /v1/rags                — Create a RAG (KB) with initial files/URLs
GET    /v1/rags                — List customer's RAGs
GET    /v1/rags/{id}           — Get RAG details
DELETE /v1/rags/{id}           — Soft-delete RAG + docs + chunks (GCS untouched)
```

**Create RAG request:**

```json
POST /v1/rags
{
  "name": "Support Knowledge Base",
  "description": "Customer support documentation",
  "file_ids": [
    "file-uuid-1",
    "file-uuid-2"
  ],
  "urls": [
    "https://docs.example.com/guide",
    "https://example.com/faq.html"
  ]
}
```

- `name` is required. `description`, `file_ids`, and `urls` are optional.
- `customer_id` comes from auth token (not in body).
- `file_ids` reference existing files in bin-storage-manager. All file_ids are validated before the RAG is created — if any file_id is invalid, the entire request fails.
- `urls` are fetched, stored in GCS via storage-manager, then processed.
- For each file/URL, a `rag_documents` row is created and async processing begins.
- Document `name` defaults to the original filename (from storage-manager) for file-based docs, or the URL for URL-based docs. Customers can override via an explicit name.
- Returns **201 Created** with the RAG resource. Document processing runs asynchronously in the background.

### Document Management

```
POST   /v1/documents           — Add more files/URLs to an existing RAG
GET    /v1/documents           — List documents (supports ?rag_id= filter)
GET    /v1/documents/{id}      — Get document details + processing status
DELETE /v1/documents/{id}      — Soft-delete document + chunks (GCS untouched)
```

**Add documents request:**

```json
POST /v1/documents
{
  "rag_id": "rag-uuid",
  "file_ids": [
    "file-uuid-3"
  ],
  "urls": [
    "https://example.com/new-page"
  ]
}
```

- Returns **202 Accepted**. Document processing runs asynchronously.
- `GET /v1/documents` supports `?rag_id=uuid` query parameter to filter by RAG. Without it, returns all documents across all customer's RAGs.

### Query

```
POST   /v1/query               — Query a RAG's knowledge base (rag_id in body)
```

**Query request/response:**

```json
// Request
{
  "rag_id": "uuid",
  "query": "How do I transfer a call?",
  "top_k": 5
}

// Response
{
  "answer": "To transfer a call, you can use...",
  "sources": [
    {
      "document_id": "uuid",
      "document_name": "call-guide.pdf",
      "section_title": "Call Transfer",
      "relevance_score": 0.91
    }
  ]
}
```

### Retired Endpoints

The following internal-only endpoints from the current implementation are retired:

```
POST /v1/rags/query              — Replaced by POST /v1/query
POST /v1/rags/index              — Replaced by document upload processing pipeline
POST /v1/rags/index/incremental  — Replaced by document upload processing pipeline
GET  /v1/rags/index/status       — Replaced by document status field
```

## File Storage

- **Raw files** are stored in GCS via `bin-storage-manager` — bin-rag-manager never touches GCS directly
- `storage_file_id` on `rag_documents` references the storage-manager resource
- **Delete behavior**: Soft-delete table rows only. GCS files are never deleted when RAGs or documents are deleted. Files are managed independently by storage-manager.

## Processing Pipeline

Documents are processed asynchronously. The API returns immediately and processing runs in the background.

### File-based documents (file_ids)

```
POST /v1/rags or POST /v1/documents with file_ids
  → Validate file_ids exist in storage-manager
  → Create rag_documents rows (status: pending, doc_type: uploaded)
  → Return 201 (POST /v1/rags) or 202 (POST /v1/documents)
  → Async background worker per document:
      1. Download file content from storage-manager
      2. Extract text using format-specific Go-native extractor
      3. Chunk text using appropriate chunker
      4. Embed chunks via OpenAI (text-embedding-3-small, 1536 dims)
      5. Insert rag_chunks rows with embeddings into PostgreSQL
      6. Update rag_documents status → ready (or error + status_message)
```

### URL-based documents (urls)

```
POST /v1/rags or POST /v1/documents with urls
  → Create rag_documents rows (status: pending, doc_type: url, source_url set)
  → Return 201 (POST /v1/rags) or 202 (POST /v1/documents)
  → Async background worker per URL:
      1. Fetch URL content
      2. Store fetched content in GCS via storage-manager → set storage_file_id
      3. Extract text (HTML stripping or format-specific)
      4. Chunk → embed → insert rag_chunks
      5. Update rag_documents status → ready (or error + status_message)
```

## Supported File Formats

All extraction is Go-native. No external services required.

| Format | Extraction Method |
|--------|------------------|
| .txt | Direct read |
| .md | Direct read + MarkdownChunker |
| .rst | Direct read + RSTChunker |
| .pdf | Go library (ledongthuc/pdf) |
| .yaml/.yml | Direct read + OpenAPIChunker |
| .docx | Go library (nguyenthenguyen/docx or similar) |
| .csv | Go stdlib encoding/csv |
| .html | Go stdlib html + strip tags |

## Query Flow

```
POST /v1/query { rag_id, query, top_k }
  1. Embed query text via OpenAI (text-embedding-3-small)
  2. Vector search:
     SELECT id, text, section_title, document_id,
            embedding <=> $query_vector AS distance
     FROM rag_chunks
     WHERE rag_id = $rag_id AND tm_delete IS NULL
     ORDER BY embedding <=> $query_vector
     LIMIT $top_k
  3. Generate answer via OpenAI LLM (gpt-4o, temperature 0.1)
     with retrieved chunks as context
  4. Return answer + source citations
```

## AI Talk Integration

RAG is consumed **within the existing AI talk action** — no separate flow action type.

- AI config gains two fields: `use_rag` (bool) and `rag_id` (UUID)
- When `use_rag` is enabled and `rag_id` is valid, the AI talk queries the customer's RAG to ground its responses during conversation
- Transparent to flow design — customer enables it in AI config, AI talk handles the rest
- The AI talk queries the RAG whenever it needs context to answer a question

## Service Architecture

```
bin-api-manager (REST)
       │
       │ RabbitMQ RPC
       ▼
bin-rag-manager
  ├── raghandler       — Core orchestration (query, document processing)
  ├── dbhandler        — PostgreSQL + pgvector operations
  ├── chunker          — Text extraction + chunking (per format)
  ├── embedder         — OpenAI embedding API client
  ├── retriever        — Vector search via PostgreSQL
  └── generator        — LLM answer generation

Connections:
  ├── PostgreSQL 17 (Cloud SQL + pgvector) — vectors + metadata
  ├── OpenAI API — embeddings (text-embedding-3-small) + LLM (gpt-4o)
  └── bin-storage-manager (via RPC) — file storage in GCS
```

- PostgreSQL client is isolated to bin-rag-manager
- Other monorepo services continue using MySQL via bin-common-handler
- RabbitMQ RPC for inter-service communication (no change from current pattern)

## Delete Behavior

| Action | rag_rags | rag_documents | rag_chunks | GCS files |
|--------|----------|---------------|------------|-----------|
| DELETE /v1/rags/{id} | Soft-delete | Soft-delete | Soft-delete | Untouched |
| DELETE /v1/documents/{id} | No change | Soft-delete | Soft-delete | Untouched |

## PostgreSQL: First-in-Monorepo Deviation

This design introduces PostgreSQL as a **new database type** in the monorepo. All other services use MySQL via `bin-common-handler`. This is a conscious trade-off — pgvector on PostgreSQL provides purpose-built vector search capabilities that MySQL cannot match.

### Implications

- **Separate Alembic migration environment**: New directory `bin-dbscheme-manager/rag-manager/` with its own `alembic.ini` targeting PostgreSQL. Existing MySQL migrations in `bin-dbscheme-manager/bin-manager/` are unaffected.
- **Custom dbhandler**: bin-rag-manager implements its own PostgreSQL-specific `dbhandler` using `github.com/lib/pq`. It cannot use `bin-common-handler`'s MySQL DBHandler or `commondatabasehandler` utilities (`PrepareFields`, `ScanRow`, etc.). However, it should follow the same architectural patterns (squirrel query builder, `DBHandler` interface, `db:` tags on model structs).
- **Operational**: Two database types to manage, monitor, and backup. PostgreSQL is already available via GCP Cloud SQL.

## OpenAPI and API Manager Integration

New API resources require updates across multiple services:

1. **Define OpenAPI schemas** in `bin-openapi-manager/openapi/openapi.yaml` for: `RagManagerRag`, `RagManagerDocument`, `RagManagerQuery`, `RagManagerQueryResponse`
2. **Define WebhookMessage structs** in bin-rag-manager `models/` for all API-facing entities (strip internal fields like denormalized `rag_id` on chunks)
3. **Regenerate OpenAPI types**: `cd bin-openapi-manager && go generate ./...`
4. **Regenerate API server code**: `cd bin-api-manager && go generate ./...`
5. **Add servicehandler methods** in `bin-api-manager` for routing HTTP requests to bin-rag-manager via RabbitMQ RPC

## Future Optimizations

- **HNSW index partitioning**: Current HNSW index covers all embeddings. As customer count grows, consider partial indexes per `rag_id` or composite filtering strategies for better query performance.
- **URL re-fetching**: Periodically re-fetch URL-sourced documents to keep content up to date.

## Not in Initial Scope

- Configurable embedding models (locked to text-embedding-3-small, 1536 dims)
- Google Document AI or advanced PDF extraction
- Separate RAG flow action type
- Document sharing across RAGs (upload separately per RAG)
- Horizontal scaling of rag-manager (single replica initially)
- URL content refresh / scheduled re-indexing
