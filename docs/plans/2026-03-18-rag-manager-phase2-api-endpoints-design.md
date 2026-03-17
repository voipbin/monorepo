# Rag Manager Phase 2: API Endpoints Design

## Problem

bin-rag-manager has all foundational infrastructure (models, dbhandler, embedder, migrations) from Phase 1 but no working API endpoints. The listenhandler returns 404 for all requests, and raghandler methods are stubs returning "not implemented". pipecat-manager needs the query endpoint to retrieve relevant document chunks for AI conversations.

## Endpoints

All endpoints are internal RabbitMQ RPC (not HTTP). They follow the monorepo's standard listenhandler routing pattern with regex-based URI matching.

### Rag CRUD

| Method | URI | Description |
|--------|-----|-------------|
| POST | `/v1/rags` | Create a new rag |
| GET | `/v1/rags/<id>` | Get a single rag |
| GET | `/v1/rags?page_size=&page_token=` | List rags (body filters: `customer_id`, `deleted`) |
| DELETE | `/v1/rags/<id>` | Delete rag + cascade documents + chunks |

### Document CRUD

| Method | URI | Description |
|--------|-----|-------------|
| POST | `/v1/documents` | Create document (`rag_id` in body, triggers async ingestion) |
| GET | `/v1/documents/<id>` | Get a single document |
| GET | `/v1/documents?page_size=&page_token=` | List documents (body filters: `customer_id`, `rag_id`, `status`, `deleted`) |
| DELETE | `/v1/documents/<id>` | Delete document + cascade chunks |

### Query

| Method | URI | Description |
|--------|-----|-------------|
| POST | `/v1/query` | Vector similarity search (requires `rag_id` + `query` in body) |

## Key Design Decisions

### 1. Query requires rag_id (scoped search)

Queries are scoped to a single rag — no cross-rag search. The `rag_id` is required in the request body. This matches the raghandler interface: `QueryRag(ctx, ragID, queryText, topK)`.

### 2. query.Response has no Answer field

rag-manager returns sources only (matching chunks with relevance scores). Answer generation is handled by the caller (pipecat-manager uses sources as AI context). The `Answer` field will be removed from `query.Response`.

### 3. Documents are top-level resources

Documents use `/v1/documents` (not nested under `/v1/rags/<id>/documents`) for consistency with the monorepo pattern. The `rag_id` is passed in the request body for creation and as a body filter for listing.

### 4. Async document ingestion

Document creation returns immediately with `status: pending`. A background goroutine handles:
1. Fetch content (from storage file or URL)
2. Parse and chunk the content
3. Embed chunks via Vertex AI
4. Store chunks + embeddings in pgvector
5. Update document status to `ready` or `error`

### 5. Filter-based listing follows monorepo pattern

List endpoints use:
- Pagination from query string (`page_size`, `page_token`)
- Filters from request body (JSON map parsed by `utilhandler.ParseFiltersFromRequestBody`)
- Type-safe conversion via `utilhandler.ConvertFilters` with `FieldStruct` definitions
- Database application via local PostgreSQL filter loop in dbhandler (not `commondatabasehandler.ApplyFields` which is MySQL-specific)

## Changes Required

### bin-rag-manager

**pkg/listenhandler/**
- `main.go` — Add regex patterns and route matching in `processRequest()`
- `v1_rags.go` — Handlers for rag CRUD (create, get, list, delete)
- `v1_documents.go` — Handlers for document CRUD (create, get, list, delete)
- `v1_query.go` — Handler for query endpoint

**pkg/raghandler/**
- `rag.go` — Implement RagCreate, RagGet, RagGetsByCustomerID (now List with filters), RagDelete
- `document.go` — Implement DocumentCreate (with async ingestion), DocumentGet, DocumentGetsByRagID (now List with filters), DocumentDelete
- `query.go` — Implement QueryRag (embed query → vector search → return sources)
- `main.go` — Update interface: list methods accept filters + pagination instead of hardcoded params

**pkg/dbhandler/**
- `rag.go` — Update RagGetsByCustomerID → RagList with filters + pagination
- `document.go` — Update DocumentGetsByRagID/CustomerID → DocumentList with filters + pagination
- `main.go` — Update interface to match

**models/rag/**
- `filters.go` — Add FieldStruct with `filter:` tags for list filtering

**models/document/**
- `filters.go` — Add FieldStruct with `filter:` tags for list filtering

**models/query/**
- `main.go` — Remove `Answer` field from `Response`

### bin-common-handler

**pkg/requesthandler/**
- `rag_rags.go` — Update `RagV1RagQuery`: add `ragID` param, change URI to `/v1/query`, remove `docTypes` param
- `main.go` — Update `RequestHandler` interface signature

## Implementation Order

1. Models: Add FieldStruct filters, remove Answer from query.Response
2. DBHandler: Update list methods to use filters + pagination
3. RagHandler: Implement all business logic methods
4. ListenHandler: Add routes and handler methods
5. bin-common-handler: Update RagV1RagQuery caller signature
6. Verification: Full workflow on all services

## Notes

- bin-api-manager does NOT expose RAG endpoints externally. RAG is an internal service consumed by other managers (e.g., pipecat-manager) via RabbitMQ RPC through bin-common-handler's requesthandler.
- The OpenAPI spec does not include RAG endpoints — they are internal-only.
