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

## External API Endpoints

bin-api-manager exposes RAG management endpoints as external HTTP APIs for customer use. The query endpoint remains internal-only.

### Rags

| Method | URI | Description |
|--------|-----|-------------|
| POST | `/rags` | Create rag (`customer_id`, `name`, `description`) |
| GET | `/rags/{id}` | Get single rag |
| GET | `/rags` | List rags (pagination, filter by `customer_id`) |
| PUT | `/rags/{id}` | Update rag (`name`, `description`) |
| DELETE | `/rags/{id}` | Delete rag + cascade documents + chunks |

### Rag Documents

| Method | URI | Description |
|--------|-----|-------------|
| POST | `/rag-documents` | Create document (`rag_id`, `name`, `doc_type`, etc.) |
| GET | `/rag-documents/{id}` | Get single document |
| GET | `/rag-documents` | List documents (filter by `rag_id`, `customer_id`) |
| DELETE | `/rag-documents/{id}` | Delete document + cascade chunks |

No PUT for rag-documents — documents are immutable after creation.

No external query endpoint — query stays internal-only via RabbitMQ RPC.

### Authorization

- Permission: `PermissionCustomerAdmin` only (no PermissionCustomerManager)
- Verify `resource.CustomerID == agent.CustomerID` for all operations
- Creates use `agent.CustomerID` as the `customer_id`
- Lists filter by `agent.CustomerID`

### Data Flow

api-manager HTTP → RabbitMQ RPC → bin-rag-manager (direct delegation, same pattern as other services).

Flat routes `/rags` and `/rag-documents` map directly to internal RPC routes `/v1/rags` and `/v1/documents`. api-manager passes through with minimal translation.

### Response Shapes

Responses use `WebhookMessage` structs (not internal model structs).

**Rag (`rag.WebhookMessage`):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "7d4e8f2a-1b3c-4d5e-9f6a-8b7c6d5e4f3a",
  "name": "Customer Support KB",
  "description": "Knowledge base for customer support conversations",
  "tm_create": "2026-03-18T10:30:00Z",
  "tm_update": "2026-03-18T10:30:00Z"
}
```

**Rag Document (`document.WebhookMessage`):**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "customer_id": "7d4e8f2a-1b3c-4d5e-9f6a-8b7c6d5e4f3a",
  "rag_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "FAQ Document",
  "doc_type": "text",
  "storage_file_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "source_url": "",
  "status": "ready",
  "status_message": "",
  "tm_create": "2026-03-18T11:00:00Z",
  "tm_update": "2026-03-18T11:15:00Z"
}
```

**List responses** use standard pagination envelope:
```json
{
  "result": [ ... ],
  "next_page_token": "2026-03-18T10:30:00Z"
}
```

## External API Changes Required

### bin-rag-manager

**pkg/raghandler/**
- `main.go` — Add `RagUpdate(ctx, id, fields)` to interface
- `rag.go` — Implement `RagUpdate`

**pkg/listenhandler/**
- `main.go` — Add PUT route regex and handler dispatch for `/v1/rags/<id>`
- `v1_rags.go` — Add `processV1RagsIDPut` handler

### bin-common-handler

**pkg/requesthandler/**
- `rag_rags.go` — Add RPC caller methods: `RagV1RagCreate`, `RagV1RagGet`, `RagV1RagGets`, `RagV1RagUpdate`, `RagV1RagDelete`
- `rag_documents.go` — Add RPC caller methods: `RagV1DocumentCreate`, `RagV1DocumentGet`, `RagV1DocumentGets`, `RagV1DocumentDelete`
- `main.go` — Update `RequestHandler` interface with new methods

### bin-openapi-manager

**openapi/openapi.yaml**
- Add `RagManagerRag` schema (matching `rag.WebhookMessage`)
- Add `RagManagerRagDocument` schema (matching `document.WebhookMessage`)
- Add `RagManagerRagDocumentDocType` enum
- Add `RagManagerRagDocumentStatus` enum
- Add path references for `/rags` and `/rag-documents` endpoints

**openapi/paths/**
- `rags/main.yaml` — GET (list) + POST (create)
- `rags/id.yaml` — GET + PUT + DELETE
- `rag-documents/main.yaml` — GET (list) + POST (create)
- `rag-documents/id.yaml` — GET + DELETE

### bin-api-manager

**server/**
- `rags.go` — HTTP handlers for rag endpoints
- `rag_documents.go` — HTTP handlers for rag-document endpoints

**pkg/servicehandler/**
- `rag.go` — ServiceHandler methods for rags (permission checks, WebhookMessage conversion)
- `rag_document.go` — ServiceHandler methods for rag-documents
- `main.go` — Add methods to `ServiceHandler` interface

## Implementation Order

1. Models: Add FieldStruct filters, remove Answer from query.Response
2. DBHandler: Update list methods to use filters + pagination
3. RagHandler: Implement all business logic methods + add RagUpdate
4. ListenHandler: Add routes and handler methods (including PUT for rags)
5. bin-common-handler: Add requesthandler RPC caller methods
6. bin-openapi-manager: Add schemas and paths
7. bin-api-manager: Add server handlers and servicehandler methods
8. Verification: Full workflow on all services

## Notes

- The query endpoint (`POST /v1/query`) remains internal-only via RabbitMQ RPC, consumed by pipecat-manager.
- The OpenAPI spec includes rag and rag-document endpoints but NOT query.
