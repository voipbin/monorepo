# Remove /rag-documents External API

**Date:** 2026-03-20
**Status:** Approved

## Problem

The `/rag-documents` endpoints (`GET /rag-documents`, `GET /rag-documents/{id}`) expose internal document implementation details to customers with no proven need. The `GET /rags/{id}` response already includes a `sources[]` array with all operationally important fields (`status`, `status_message`, `storage_file_id`, `source_url`).

Keeping these endpoints adds maintenance overhead across 4 services (bin-openapi-manager, bin-api-manager, bin-common-handler, bin-rag-manager) and exposes internal fields like `doc_type`, `retry_count`, and document-level timestamps.

## Decision

Remove all customer-facing `/rag-documents` endpoints. Internal document handling in bin-rag-manager stays untouched.

## Data Gap (Accepted)

Customers lose access to 6 fields: `id`, `name`, `doc_type`, `customer_id`, `rag_id`, `tm_create`/`tm_update`. All are either redundant (derivable from the parent RAG) or internal. The `doc_type` is implicitly conveyed by which `sources[]` field is populated (file ID = upload, URL = url fetch).

## Files to DELETE (10)

1. `bin-openapi-manager/openapi/paths/rag-documents/main.yaml` — Path spec for GET /rag-documents
2. `bin-openapi-manager/openapi/paths/rag-documents/id.yaml` — Path spec for GET /rag-documents/{id}
3. `bin-api-manager/server/rag_documents.go` — HTTP handlers
4. `bin-api-manager/server/rag_documents_test.go` — HTTP handler tests
5. `bin-api-manager/pkg/servicehandler/rag_document.go` — Service logic
6. `bin-api-manager/pkg/servicehandler/rag_document_test.go` — Service logic tests
7. `bin-api-manager/docsdev/source/rag_struct_document.rst` — RST docs for document struct
8. `bin-common-handler/pkg/requesthandler/rag_documents.go` — RPC methods
9. `bin-rag-manager/pkg/listenhandler/v1_documents.go` — RPC route handlers
10. `bin-rag-manager/models/document/webhook.go` — WebhookMessage (only used for external API)

## Files to MODIFY (7)

1. `bin-openapi-manager/openapi/openapi.yaml` — Remove `RagManagerRagDocument` and `RagManagerRagDocumentDocType` schemas + path references. **KEEP `RagManagerRagDocumentStatus`** (used by `RagManagerRag` and `RagManagerRagSource`).
2. `bin-api-manager/pkg/servicehandler/main.go` — Remove `RagDocumentGet`/`RagDocumentGets` from interface, remove `rmdocument` import.
3. `bin-api-manager/docsdev/source/rag.rst` — Remove `.. include:: rag_struct_document.rst`.
4. `bin-api-manager/docsdev/source/rag_overview.rst` — Rewrite 2 references to `/rag-documents` to use `GET /rags/{id}` sources.
5. `bin-api-manager/docsdev/source/rag_tutorial.rst` — Remove/rewrite individual document status section and troubleshooting references.
6. `bin-common-handler/pkg/requesthandler/main.go` — Remove `RagV1DocumentGet`/`RagV1DocumentGets` from interface, remove `rmdocument` import.
7. `bin-rag-manager/pkg/listenhandler/main.go` — Remove document regex patterns and switch cases.

## What Stays Untouched

- `bin-rag-manager/models/document/main.go` — internal model used by raghandler
- `bin-rag-manager/pkg/dbhandler/document.go` — internal DB operations
- `bin-rag-manager/pkg/raghandler/` — ingestion pipeline
- `bin-rag-manager/models/rag/webhook.go` — RAG webhook with Sources[] (imports rmdocument.Status)
- `RagManagerRagDocumentStatus` schema in openapi.yaml
- All `/rags` endpoints
- `bin-rag-manager/pkg/listenhandler/main_test.go` — mock serves internal RagHandler interface

## Verification (Dependency Order)

1. Complete all 10 deletions and 7 modifications.
2. `cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./...`
3. `cd bin-common-handler && go mod tidy && go mod vendor && go generate ./...`
4. `cd bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
5. `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
6. `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` then `git add -f bin-api-manager/docsdev/build/`

## Risk

Low. Read-only endpoints with no known consumer. RAG feature is new (PR #706 merged 2026-03-19). No CI/deployment/monitoring references.
