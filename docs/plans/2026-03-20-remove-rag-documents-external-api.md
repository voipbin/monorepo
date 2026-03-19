# Remove /rag-documents External API — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove the customer-facing `/rag-documents` endpoints while preserving internal document handling in bin-rag-manager.

**Architecture:** Deletion of external API surface across 4 services (OpenAPI spec, API gateway, shared RPC library, service listen handler). Internal document model, DB operations, and ingestion pipeline remain untouched.

**Tech Stack:** Go, OpenAPI 3.0 (oapi-codegen), RabbitMQ RPC, Sphinx RST docs

**Design doc:** `docs/plans/2026-03-20-remove-rag-documents-external-api-design.md`

---

### Task 1: Delete files and modify OpenAPI spec (bin-openapi-manager)

**Files:**
- Delete: `bin-openapi-manager/openapi/paths/rag-documents/main.yaml`
- Delete: `bin-openapi-manager/openapi/paths/rag-documents/id.yaml`
- Modify: `bin-openapi-manager/openapi/openapi.yaml`

**Step 1: Delete the path YAML files**

```bash
rm bin-openapi-manager/openapi/paths/rag-documents/main.yaml
rm bin-openapi-manager/openapi/paths/rag-documents/id.yaml
rmdir bin-openapi-manager/openapi/paths/rag-documents
```

**Step 2: Remove path references from openapi.yaml (lines 6911-6914)**

Remove these 4 lines:
```yaml
  /rag-documents/{id}:
    $ref: './paths/rag-documents/id.yaml'
  /rag-documents:
    $ref: './paths/rag-documents/main.yaml'
```

**Step 3: Remove `RagManagerRagDocument` schema from openapi.yaml (lines 5198-5247)**

Delete the entire `RagManagerRagDocument:` block (from line 5198 through line 5247, ending before the blank line).

**Step 4: Remove `RagManagerRagDocumentDocType` schema from openapi.yaml (lines 5249-5267)**

Delete the entire `RagManagerRagDocumentDocType:` block.

**CRITICAL: DO NOT delete `RagManagerRagDocumentStatus` (lines 5269-5287).** It is referenced by `RagManagerRag.status` and `RagManagerRagSource.status`.

**Step 5: Run verification for bin-openapi-manager**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./...
```

Expected: `gens/models/gen.go` regenerated without `RagManagerRagDocument` and `RagManagerRagDocumentDocType` types. `RagManagerRagDocumentStatus` still present.

---

### Task 2: Remove RPC methods from bin-common-handler

**Files:**
- Delete: `bin-common-handler/pkg/requesthandler/rag_documents.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go`

**Step 1: Delete the RPC implementation file**

```bash
rm bin-common-handler/pkg/requesthandler/rag_documents.go
```

**Step 2: Remove interface methods from main.go (lines 1326-1328)**

Remove these lines from the `RequestHandler` interface:
```go
	// rag-manager documents
	RagV1DocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error)
	RagV1DocumentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmdocument.Field]any) ([]*rmdocument.Document, error)
```

**Step 3: Remove the `rmdocument` import from main.go (line 99)**

Remove this import line:
```go
	rmdocument "monorepo/bin-rag-manager/models/document"
```

**Step 4: Run verification for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./...
```

Expected: `mock_main.go` regenerated without `RagV1DocumentGet`/`RagV1DocumentGets` mock methods.

---

### Task 3: Remove listen handler routes from bin-rag-manager

**Files:**
- Delete: `bin-rag-manager/pkg/listenhandler/v1_documents.go`
- Delete: `bin-rag-manager/models/document/webhook.go`
- Modify: `bin-rag-manager/pkg/listenhandler/main.go`

**Step 1: Delete the document route handler and webhook files**

```bash
rm bin-rag-manager/pkg/listenhandler/v1_documents.go
rm bin-rag-manager/models/document/webhook.go
```

**Step 2: Remove document regex patterns from main.go (lines 34-36)**

Remove these lines:
```go
	// document routes
	regV1Documents   = regexp.MustCompile(`^/v1/documents(\?.*)?$`)
	regV1DocumentsID = regexp.MustCompile(`^/v1/documents/` + regUUID + `(\?.*)?$`)
```

**Step 3: Remove document switch cases from main.go (lines 132-139)**

Remove these lines from the `processRequest` switch statement:
```go
	// document routes — read-only (POST and DELETE removed)
	case regV1DocumentsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DocumentsIDGet(ctx, m)
		requestType = "/v1/documents/<document-id>"

	case regV1Documents.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DocumentsGet(ctx, m)
		requestType = "/v1/documents"
```

**Step 4: Run full verification for bin-rag-manager**

```bash
cd bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass. No compile errors.

---

### Task 4: Remove API handlers and service logic from bin-api-manager

**Files:**
- Delete: `bin-api-manager/server/rag_documents.go`
- Delete: `bin-api-manager/server/rag_documents_test.go`
- Delete: `bin-api-manager/pkg/servicehandler/rag_document.go`
- Delete: `bin-api-manager/pkg/servicehandler/rag_document_test.go`
- Modify: `bin-api-manager/pkg/servicehandler/main.go`

**Step 1: Delete all 4 files**

```bash
rm bin-api-manager/server/rag_documents.go
rm bin-api-manager/server/rag_documents_test.go
rm bin-api-manager/pkg/servicehandler/rag_document.go
rm bin-api-manager/pkg/servicehandler/rag_document_test.go
```

**Step 2: Remove interface methods from servicehandler/main.go (lines 917-920)**

Remove these lines from the `ServiceHandler` interface:
```go
	// RAG Documents (read-only)
	RagDocumentGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdocument.WebhookMessage, error)
	RagDocumentGets(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, size uint64, token string) ([]*rmdocument.WebhookMessage, error)
```

**Step 3: Remove the `rmdocument` import from servicehandler/main.go (line 77)**

Remove this import line:
```go
	rmdocument "monorepo/bin-rag-manager/models/document"
```

Verify `rmrag` import on line 76 is still present (needed by RAG endpoints).

**Step 4: Run full verification for bin-api-manager**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All tests pass. Generated server code in `gens/openapi_server/gen.go` no longer contains `GetRagDocuments`/`GetRagDocumentsId` methods. Mock in `mock_main.go` no longer contains `RagDocumentGet`/`RagDocumentGets`.

---

### Task 5: Update RST documentation

**Files:**
- Delete: `bin-api-manager/docsdev/source/rag_struct_document.rst`
- Modify: `bin-api-manager/docsdev/source/rag.rst` (line 12)
- Modify: `bin-api-manager/docsdev/source/rag_overview.rst` (lines 10, 121)
- Modify: `bin-api-manager/docsdev/source/rag_tutorial.rst` (lines 152-195, 284, 288)

**Step 1: Delete the document struct RST file**

```bash
rm bin-api-manager/docsdev/source/rag_struct_document.rst
```

**Step 2: Remove the include from rag.rst (line 12)**

Remove this line:
```rst
.. include:: rag_struct_document.rst
```

**Step 3: Update rag_overview.rst**

Line 10 — Change:
```
...or individual document status via ``GET https://api.voipbin.net/v1.0/rag-documents/{id}``.
```
To:
```
...Check the ``sources`` array in the RAG response for per-document status.
```

Line 121 — Change:
```
...poll ``GET /rags/{id}`` to check the RAG's ``status`` field, or poll ``GET /rag-documents/{id}`` for individual document status.
```
To:
```
...poll ``GET /rags/{id}`` to check the RAG's ``status`` field. Each entry in the ``sources`` array shows per-document ``status`` and ``status_message``.
```

**Step 4: Update rag_tutorial.rst**

Remove the entire "Step 4: Check Individual Document Status" section (lines 152-195). Renumber subsequent steps: Step 5 → Step 4, Step 6 → Step 5.

Line 284 — Change:
```
Verify the UUID was obtained from a recent ``GET /rags`` or ``GET /rag-documents`` call.
```
To:
```
Verify the UUID was obtained from a recent ``GET /rags`` call.
```

Line 288 — Change:
```
Check individual source statuses in the RAG response or via ``GET /rag-documents``.
```
To:
```
Check individual source statuses in the ``sources`` array of the ``GET /rags/{id}`` response.
```

**Step 5: Rebuild Sphinx HTML**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

Expected: Clean build with no warnings about missing includes or references.

**Step 6: Stage the built HTML**

```bash
git add -f bin-api-manager/docsdev/build/
```

---

### Task 6: Final verification and commit

**Step 1: Verify all services build cleanly**

Run verification across all 4 services one final time to catch any missed dependencies:

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./...
cd ../bin-common-handler && go mod tidy && go mod vendor && go generate ./...
cd ../bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass with zero errors.

**Step 2: Review the diff**

```bash
git diff --stat
git diff
```

Verify:
- 10 files deleted
- 7 files modified (openapi.yaml, 2x main.go interfaces, listenhandler/main.go, 3 RST files)
- Generated files updated (gen.go, mock_main.go in each service)
- No unintended changes

**Step 3: Commit**

```bash
git add -A
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Remove-rag-documents-external-api

Remove customer-facing /rag-documents endpoints. Customers use GET /rags/{id}
with embedded sources[] for document status instead.

- bin-openapi-manager: Remove RagManagerRagDocument and RagManagerRagDocumentDocType schemas, keep RagManagerRagDocumentStatus
- bin-openapi-manager: Remove /rag-documents path references
- bin-api-manager: Remove rag_documents server handlers and servicehandler logic
- bin-api-manager: Remove RagDocumentGet/RagDocumentGets from ServiceHandler interface
- bin-api-manager: Update RST docs to remove /rag-documents references, rebuild HTML
- bin-common-handler: Remove RagV1DocumentGet/RagV1DocumentGets from RequestHandler interface
- bin-rag-manager: Remove document listen handler routes and WebhookMessage
- docs: Add design document for rag-documents API removal"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-Remove-rag-documents-external-api
```

Then create PR with title matching branch name.
