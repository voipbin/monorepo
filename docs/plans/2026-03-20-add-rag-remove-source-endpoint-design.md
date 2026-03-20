# Design: Add DELETE /v1/rags/{rag-id}/sources/{source-id} Endpoint

**Date:** 2026-03-20
**Branch:** NOJIRA-Add-rag-remove-source-endpoint

## Problem

There is no API to remove an individual source from a RAG. Users can add sources via `POST /v1/rags/{id}/sources` but cannot remove them individually. Additionally, the `Source` struct in the API response has no `id` field, so users cannot reference specific sources.

## Approach

### 1. Add `commonidentity.Identity` to `Source` struct

Embed `commonidentity.Identity` (provides `id` and `customer_id`) into `rag.Source`, consistent with how other model structs like `Outdial` use it. Populate from the document's ID and CustomerID when building the sources list in raghandler.

### 2. Add DELETE endpoint

**Endpoint:** `DELETE /v1/rags/{rag-id}/sources/{source-id}`

**Behavior:** Soft-deletes a single source (document) and its chunks. Returns the updated RAG with refreshed `sources[]` list (consistent with `RagAddSources` return pattern).

**Flow:**
1. `bin-api-manager` servicehandler `RagRemoveSource()` — permission check, calls RPC
2. `bin-common-handler` requesthandler `RagV1RagRemoveSource()` — sends DELETE via RabbitMQ
3. `bin-rag-manager` listenhandler — new regex route + handler, extracts both UUIDs
4. `bin-rag-manager` raghandler `RagRemoveSource(ctx, ragID, sourceID)`:
   - Fetches document, validates it belongs to the RAG
   - `ChunkSoftDeleteByDocumentID(sourceID)` — already exists in dbhandler
   - `DocumentDelete(sourceID)` — already exists in dbhandler
   - Returns refreshed RAG via `RagGet(ragID)`

**Error cases:**
- 404 if RAG not found
- 404 if source not found or doesn't belong to this RAG
- CustomerAdmin permission required

### 3. Update OpenAPI spec

Add `id` and `customer_id` fields to `RagManagerRagSource` schema. Add new DELETE path for `/rags/{rag-id}/sources/{source-id}`.

## Files to Change

| File | Change |
|------|--------|
| `bin-rag-manager/models/rag/main.go` | Embed `commonidentity.Identity` in `Source` struct |
| `bin-rag-manager/pkg/raghandler/rag.go` | Populate Source identity; add `RagRemoveSource()` |
| `bin-rag-manager/pkg/raghandler/main.go` | Add `RagRemoveSource` to interface |
| `bin-rag-manager/pkg/listenhandler/main.go` | Add regex route for DELETE sources/{id} |
| `bin-rag-manager/pkg/listenhandler/v1_rags.go` | Add `processV1RagsIDSourcesIDDelete()` handler |
| `bin-common-handler/pkg/requesthandler/main.go` | Add `RagV1RagRemoveSource` to interface |
| `bin-common-handler/pkg/requesthandler/rag_rags.go` | Implement `RagV1RagRemoveSource()` |
| `bin-api-manager/pkg/servicehandler/main.go` | Add `RagRemoveSource` to interface |
| `bin-api-manager/pkg/servicehandler/rag.go` | Implement `RagRemoveSource()` |
| `bin-openapi-manager/openapi/paths/rags/id_sources_id.yaml` | New DELETE path |
| `bin-openapi-manager/openapi/openapi.yaml` | Reference new path file |
| `bin-openapi-manager/openapi/schemas/rag_manager.yaml` | Add id/customer_id to RagManagerRagSource |

## Decisions

- **Allow deleting processing sources** — consistent with `RagDelete` which cascades regardless of status
- **Return updated RAG** — consistent with `RagAddSources` return pattern
- **`ChunkSoftDeleteByDocumentID` already exists** — no new DB method needed
- **Source ID = Document ID** — the internal document ID is exposed as the source ID
