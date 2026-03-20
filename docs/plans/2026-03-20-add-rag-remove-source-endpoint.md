# Add DELETE /v1/rags/{rag-id}/sources/{source-id} Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a DELETE endpoint for removing individual sources from a RAG, and add `commonidentity.Identity` to the `Source` struct so source IDs are exposed in API responses.

**Architecture:** The change spans 4 services: bin-rag-manager (model, business logic, RPC handler), bin-common-handler (inter-service RPC method), bin-api-manager (API gateway handler), and bin-openapi-manager (API contract). Each layer follows existing patterns.

**Tech Stack:** Go, RabbitMQ RPC, PostgreSQL, OpenAPI 3.0

---

### Task 1: Add `commonidentity.Identity` to `Source` struct

**Files:**
- Modify: `bin-rag-manager/models/rag/main.go:27-32`

**Step 1: Update Source struct**

Add the `commonidentity.Identity` embed and update the import:

```go
import (
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	rmdocument "monorepo/bin-rag-manager/models/document"
)

// Source represents a single source (document) in the RAG response.
type Source struct {
	commonidentity.Identity `json:",inline"`

	StorageFileID *uuid.UUID        `json:"storage_file_id,omitempty"`
	SourceURL     string            `json:"source_url,omitempty"`
	Status        rmdocument.Status `json:"status,omitempty"`
	StatusMessage string            `json:"status_message,omitempty"`
}
```

**Step 2: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`
Expected: BUILD SUCCESS

---

### Task 2: Populate Source identity in `buildSources()`

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/rag.go:242-260`

**Step 1: Update buildSources to populate Identity**

```go
func buildSources(docs []*document.Document) []rag.Source {
	sources := make([]rag.Source, 0, len(docs))
	for _, d := range docs {
		s := rag.Source{
			Status:        d.Status,
			StatusMessage: d.StatusMessage,
		}
		s.ID = d.ID
		s.CustomerID = d.CustomerID
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

**Step 2: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`
Expected: BUILD SUCCESS

---

### Task 3: Add `RagRemoveSource` to raghandler interface and implementation

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/main.go:21-36` (interface)
- Modify: `bin-rag-manager/pkg/raghandler/rag.go` (append implementation)

**Step 1: Add method to RagHandler interface**

In `main.go`, add after `RagAddSources`:

```go
RagRemoveSource(ctx context.Context, ragID, sourceID uuid.UUID) (*rag.Rag, error)
```

**Step 2: Implement RagRemoveSource**

Append to `rag.go`:

```go
func (h *ragHandler) RagRemoveSource(ctx context.Context, ragID, sourceID uuid.UUID) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "RagRemoveSource",
		"rag_id":    ragID,
		"source_id": sourceID,
	})

	// Verify document exists and belongs to this RAG
	doc, err := h.dbHandler.DocumentGet(ctx, sourceID)
	if err != nil {
		log.Errorf("Could not get document. err: %v", err)
		return nil, fmt.Errorf("could not get source: %w", err)
	}
	log.WithField("document", doc).Debugf("Retrieved document. document_id: %s", doc.ID)

	if doc.RagID != ragID {
		log.Errorf("Document does not belong to this rag. doc.rag_id: %s, rag_id: %s", doc.RagID, ragID)
		return nil, fmt.Errorf("source does not belong to this rag")
	}

	// Cascade: soft-delete chunks, then delete document
	if err := h.dbHandler.ChunkSoftDeleteByDocumentID(ctx, sourceID); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return nil, fmt.Errorf("could not delete source chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDelete(ctx, sourceID); err != nil {
		log.Errorf("Could not delete document. err: %v", err)
		return nil, fmt.Errorf("could not delete source: %w", err)
	}
	log.Debugf("Deleted source. source_id: %s", sourceID)

	return h.RagGet(ctx, ragID)
}
```

**Step 3: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`
Expected: BUILD SUCCESS

---

### Task 4: Add listenhandler route and handler

**Files:**
- Modify: `bin-rag-manager/pkg/listenhandler/main.go:26-36` (add regex)
- Modify: `bin-rag-manager/pkg/listenhandler/main.go:101-137` (add switch case)
- Modify: `bin-rag-manager/pkg/listenhandler/v1_rags.go` (append handler)

**Step 1: Add regex for `/v1/rags/{id}/sources/{id}`**

In `main.go`, add after `regV1RagsIDSources`:

```go
regV1RagsIDSourcesID = regexp.MustCompile(`^/v1/rags/` + regUUID + `/sources/` + regUUID + `(\?.*)?$`)
```

**Step 2: Add switch case in processRequest**

Add BEFORE the `regV1RagsIDSources` case (more specific route first):

```go
case regV1RagsIDSourcesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
	response, err = h.processV1RagsIDSourcesIDDelete(ctx, m)
	requestType = "/v1/rags/<rag-id>/sources/<source-id>"
```

**Step 3: Implement handler in v1_rags.go**

Append to `v1_rags.go`:

```go
func (h *listenHandler) processV1RagsIDSourcesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDSourcesIDDelete",
	})

	// URI: /v1/rags/{rag-id}/sources/{source-id}
	// Split: ["", "v1", "rags", "{rag-id}", "sources", "{source-id}"]
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		return simpleResponse(400), nil
	}

	ragID := uuid.FromStringOrNil(uriItems[3])
	if ragID == uuid.Nil {
		log.Errorf("Could not parse rag ID from URI.")
		return simpleResponse(400), nil
	}

	sourceID := uuid.FromStringOrNil(uriItems[5])
	if sourceID == uuid.Nil {
		log.Errorf("Could not parse source ID from URI.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagRemoveSource(ctx, ragID, sourceID)
	if err != nil {
		log.Errorf("Could not remove source. err: %v", err)
		return simpleResponse(500), nil
	}

	return jsonResponse(200, r), nil
}
```

**Step 4: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`
Expected: BUILD SUCCESS

---

### Task 5: Add `RagV1RagRemoveSource` to requesthandler (bin-common-handler)

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:1326` (interface)
- Modify: `bin-common-handler/pkg/requesthandler/rag_rags.go` (append implementation)

**Step 1: Add to RequestHandler interface**

After `RagV1RagAddSources`, add:

```go
RagV1RagRemoveSource(ctx context.Context, ragID, sourceID uuid.UUID) (*rmrag.Rag, error)
```

**Step 2: Implement RagV1RagRemoveSource**

Append to `rag_rags.go`:

```go
// RagV1RagRemoveSource sends a request to rag-manager to remove a source from a rag.
func (r *requestHandler) RagV1RagRemoveSource(ctx context.Context, ragID, sourceID uuid.UUID) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s/sources/%s", ragID, sourceID)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodDelete, "rag/rags.sources", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

**Step 3: Verify it compiles**

Run: `cd bin-common-handler && go build ./...`
Expected: BUILD SUCCESS

---

### Task 6: Add `RagRemoveSource` to servicehandler (bin-api-manager)

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/main.go:915` (interface)
- Modify: `bin-api-manager/pkg/servicehandler/rag.go` (append implementation)

**Step 1: Add to ServiceHandler interface**

After `RagAddSources`, add:

```go
RagRemoveSource(ctx context.Context, a *amagent.Agent, ragID, sourceID uuid.UUID) (*rmrag.WebhookMessage, error)
```

**Step 2: Implement RagRemoveSource**

Append to `rag.go`:

```go
func (h *serviceHandler) RagRemoveSource(ctx context.Context, a *amagent.Agent, ragID, sourceID uuid.UUID) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagRemoveSource",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
		"source_id":   sourceID,
	})
	log.Debug("Removing source from rag.")

	// Verify RAG exists and belongs to this customer
	tmp, err := h.ragGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	r, err := h.reqHandler.RagV1RagRemoveSource(ctx, ragID, sourceID)
	if err != nil {
		log.Errorf("Could not remove source. err: %v", err)
		return nil, err
	}

	res := r.ConvertWebhookMessage()
	return res, nil
}
```

**Step 3: Verify it compiles**

Run: `cd bin-api-manager && go build ./...`
Expected: BUILD SUCCESS (may fail until mocks are regenerated — that's OK)

---

### Task 7: Update OpenAPI spec

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (schema + path ref)
- Create: `bin-openapi-manager/openapi/paths/rags/id_sources_id.yaml` (DELETE path)

**Step 1: Add `id` and `customer_id` to RagManagerRagSource schema**

In `openapi.yaml`, update `RagManagerRagSource` (around line 5218):

```yaml
    RagManagerRagSource:
      type: object
      description: "A source document in a RAG knowledge base."
      properties:
        id:
          type: string
          format: uuid
          description: "The unique identifier of the source (document). Use this ID with `DELETE /rags/{rag-id}/sources/{source-id}` to remove the source."
          example: "c3d4e5f6-a7b8-9012-cdef-123456789012"
        customer_id:
          type: string
          format: uuid
          description: "The customer ID that owns this source. Returned from the `GET /customer` response."
          example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        storage_file_id:
          type: string
          format: uuid
          description: "The storage file ID if the source is an uploaded file. Returned from the `POST /files` response."
          example: "b2c3d4e5-f6a7-8901-bcde-f12345678901"
        source_url:
          type: string
          format: uri
          description: "The URL if the source is a web document."
          example: "https://example.com/docs/faq.html"
        status:
          $ref: '#/components/schemas/RagManagerRagDocumentStatus'
        status_message:
          type: string
          description: "Additional details about the current ingestion status."
          example: "Document parsed and 42 chunks created"
```

**Step 2: Create DELETE path file**

Create `bin-openapi-manager/openapi/paths/rags/id_sources_id.yaml`:

```yaml
delete:
  summary: Remove a source from a rag
  description: Removes a single source (document) and its chunks from a RAG knowledge base. Returns the updated RAG with refreshed sources list.
  tags:
    - RAG
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
      description: "The unique identifier of the rag. Returned from the `POST /rags` response."
    - name: source_id
      in: path
      required: true
      schema:
        type: string
        format: uuid
      description: "The unique identifier of the source to remove. Returned from the `id` field of the `sources[]` array in `GET /rags/{id}` response."
  responses:
    '200':
      description: Successfully removed source. Returns the updated RAG.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRag'
```

**Step 3: Add path reference in openapi.yaml**

In `openapi.yaml` paths section (around line 6840), add BEFORE `/rags/{id}/sources`:

```yaml
  /rags/{id}/sources/{source_id}:
    $ref: './paths/rags/id_sources_id.yaml'
```

**Step 4: Verify spec and regenerate**

Run: `cd bin-openapi-manager && go generate ./...`
Expected: SUCCESS, `gens/models/gen.go` regenerated

---

### Task 8: Regenerate mocks and run verification

**Step 1: Regenerate mocks and verify bin-rag-manager**

```bash
cd bin-rag-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS

**Step 2: Regenerate mocks and verify bin-common-handler**

```bash
cd bin-common-handler && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS

**Step 3: Regenerate mocks and verify bin-api-manager**

```bash
cd bin-api-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS

**Step 4: Verify bin-openapi-manager**

```bash
cd bin-openapi-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS

---

### Task 9: Commit, push, create PR

**Step 1: Stage all changes**

```bash
git add -A
```

**Step 2: Commit**

```
NOJIRA-Add-rag-remove-source-endpoint

Add DELETE /v1/rags/{rag-id}/sources/{source-id} endpoint to remove individual sources
from a RAG, and add id/customer_id to Source struct for source identification.

- bin-rag-manager: Embed commonidentity.Identity in Source struct
- bin-rag-manager: Populate source identity in buildSources()
- bin-rag-manager: Add RagRemoveSource to raghandler interface and implementation
- bin-rag-manager: Add listenhandler route and handler for DELETE sources/{id}
- bin-common-handler: Add RagV1RagRemoveSource to requesthandler interface and implementation
- bin-api-manager: Add RagRemoveSource to servicehandler interface and implementation
- bin-openapi-manager: Add id/customer_id to RagManagerRagSource schema
- bin-openapi-manager: Add DELETE path for /rags/{id}/sources/{source_id}
- docs: Add design document for the feature
```

**Step 3: Push and create PR**
