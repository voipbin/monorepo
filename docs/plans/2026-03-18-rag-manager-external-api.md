# Rag Manager External API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Expose RAG management via external HTTP API endpoints (`/rags` and `/rag-documents`) through bin-api-manager, with `PermissionCustomerAdmin` authorization.

**Architecture:** bin-api-manager HTTP handlers delegate to bin-rag-manager via RabbitMQ RPC through bin-common-handler's requesthandler. Flat routes `/rags` and `/rag-documents` map directly to internal RPC routes `/v1/rags` and `/v1/documents`. No external query endpoint — query stays internal-only.

**Tech Stack:** Go, OpenAPI 3.0 (oapi-codegen), gin HTTP framework, RabbitMQ RPC

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Rag-manager-phase2-api-endpoints`

**Design doc:** `docs/plans/2026-03-18-rag-manager-phase2-api-endpoints-design.md`

---

### Task 1: Add RagUpdate to raghandler and listenhandler PUT route

**Why:** The design requires `PUT /rags/{id}` to update name/description. DBHandler already has `RagUpdate`, but raghandler and listenhandler don't.

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/main.go` — add `RagUpdate` to interface
- Modify: `bin-rag-manager/pkg/raghandler/rag.go` — implement `RagUpdate`
- Modify: `bin-rag-manager/pkg/listenhandler/main.go` — add PUT route regex and switch case
- Modify: `bin-rag-manager/pkg/listenhandler/v1_rags.go` — add `processV1RagsIDPut` handler

**Step 1: Add RagUpdate to raghandler interface**

In `bin-rag-manager/pkg/raghandler/main.go`, add to the `RagHandler` interface after `RagList`:

```go
RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error)
```

**Step 2: Implement RagUpdate in rag.go**

Add to `bin-rag-manager/pkg/raghandler/rag.go`:

```go
func (h *ragHandler) RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "RagUpdate",
		"id":     id,
		"fields": fields,
	})

	if err := h.dbHandler.RagUpdate(ctx, id, fields); err != nil {
		log.Errorf("Could not update rag. err: %v", err)
		return nil, fmt.Errorf("could not update rag: %w", err)
	}

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get updated rag. err: %v", err)
		return nil, fmt.Errorf("could not get updated rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Updated rag. rag_id: %s", r.ID)

	return r, nil
}
```

**Step 3: Add PUT route to listenhandler**

In `bin-rag-manager/pkg/listenhandler/main.go`, add a PUT case to the switch (after the existing `regV1RagsID` DELETE case):

```go
case regV1RagsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
	response, err = h.processV1RagsIDPut(ctx, m)
	requestType = "/v1/rags/<rag-id>"
```

**Step 4: Implement processV1RagsIDPut handler**

Add to `bin-rag-manager/pkg/listenhandler/v1_rags.go`:

```go
func (h *listenHandler) processV1RagsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1RagsIDPut",
		"uri":  m.URI,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Could not parse the URI correctly.")
		return simpleResponse(400), nil
	}

	id, err := uuid.FromString(uriItems[3])
	if err != nil {
		log.Errorf("Could not parse the rag id. err: %v", err)
		return simpleResponse(400), nil
	}

	var req struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
	}
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not parse the request body. err: %v", err)
		return simpleResponse(400), nil
	}

	fields := map[rag.Field]any{}
	if req.Name != nil {
		fields[rag.FieldName] = *req.Name
	}
	if req.Description != nil {
		fields[rag.FieldDescription] = *req.Description
	}

	if len(fields) == 0 {
		log.Errorf("No fields to update.")
		return simpleResponse(400), nil
	}

	r, err := h.ragHandler.RagUpdate(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update the rag. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(r)
	if err != nil {
		log.Errorf("Could not marshal the response. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		Data:       data,
	}
	return res, nil
}
```

**Step 5: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`

**Step 6: Run tests**

Run: `cd bin-rag-manager && go test ./...`

**Step 7: Commit**

```
git add bin-rag-manager/pkg/raghandler/ bin-rag-manager/pkg/listenhandler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-rag-manager: Add RagUpdate to raghandler interface and implementation
- bin-rag-manager: Add PUT /v1/rags/<id> route to listenhandler"
```

---

### Task 2: Add requesthandler RPC caller methods for rags and documents

**Why:** bin-api-manager calls bin-rag-manager via requesthandler RPC methods. Currently only `RagV1RagQuery` exists. We need CRUD methods for rags and documents.

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go` — add methods to `RequestHandler` interface
- Modify: `bin-common-handler/pkg/requesthandler/rag_rags.go` — add rag RPC methods
- Create: `bin-common-handler/pkg/requesthandler/rag_documents.go` — add document RPC methods

**Step 1: Add rag RPC methods to rag_rags.go**

Add to `bin-common-handler/pkg/requesthandler/rag_rags.go` (after existing `RagV1RagQuery`). Add necessary imports: `rmrag "monorepo/bin-rag-manager/models/rag"`, `rmdocument "monorepo/bin-rag-manager/models/document"`, `"fmt"`, `"net/url"`.

```go
func (r *requestHandler) RagV1RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rmrag.Rag, error) {
	uri := "/v1/rags"

	req := struct {
		CustomerID uuid.UUID `json:"customer_id"`
		Name       string    `json:"name"`
		Description string   `json:"description"`
	}{
		CustomerID: customerID,
		Name:       name,
		Description: description,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal request")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) RagV1RagGet(ctx context.Context, id uuid.UUID) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) RagV1RagGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmrag.Field]any) ([]*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

func (r *requestHandler) RagV1RagUpdate(ctx context.Context, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.Rag, error) {
	uri := fmt.Sprintf("/v1/rags/%s", id)

	m, err := json.Marshal(fields)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal fields")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPut, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmrag.Rag
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) RagV1RagDelete(ctx context.Context, id uuid.UUID) error {
	uri := fmt.Sprintf("/v1/rags/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodDelete, "rag/rags", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if tmp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", tmp.StatusCode)
	}

	return nil
}
```

**Step 2: Create rag_documents.go for document RPC methods**

Create `bin-common-handler/pkg/requesthandler/rag_documents.go`:

```go
package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/sock"
	rmdocument "monorepo/bin-rag-manager/models/document"
)

func (r *requestHandler) RagV1DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType rmdocument.DocType, sourceURL string, storageFileID uuid.UUID) (*rmdocument.Document, error) {
	uri := "/v1/documents"

	req := struct {
		CustomerID    uuid.UUID          `json:"customer_id"`
		RagID         uuid.UUID          `json:"rag_id"`
		Name          string             `json:"name"`
		DocType       rmdocument.DocType `json:"doc_type"`
		SourceURL     string             `json:"source_url,omitempty"`
		StorageFileID uuid.UUID          `json:"storage_file_id,omitempty"`
	}{
		CustomerID:    customerID,
		RagID:         ragID,
		Name:          name,
		DocType:       docType,
		SourceURL:     sourceURL,
		StorageFileID: storageFileID,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal request")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmdocument.Document
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) RagV1DocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error) {
	uri := fmt.Sprintf("/v1/documents/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res rmdocument.Document
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

func (r *requestHandler) RagV1DocumentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmdocument.Field]any) ([]*rmdocument.Document, error) {
	uri := fmt.Sprintf("/v1/documents?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodGet, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*rmdocument.Document
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

func (r *requestHandler) RagV1DocumentDelete(ctx context.Context, id uuid.UUID) error {
	uri := fmt.Sprintf("/v1/documents/%s", id)

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodDelete, "rag/documents", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if tmp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", tmp.StatusCode)
	}

	return nil
}
```

**Step 3: Update RequestHandler interface in main.go**

Add to the `RequestHandler` interface in `bin-common-handler/pkg/requesthandler/main.go`, near the existing `RagV1RagQuery` declaration:

```go
// Rag Manager - Rags
RagV1RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rmrag.Rag, error)
RagV1RagGet(ctx context.Context, id uuid.UUID) (*rmrag.Rag, error)
RagV1RagGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmrag.Field]any) ([]*rmrag.Rag, error)
RagV1RagUpdate(ctx context.Context, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.Rag, error)
RagV1RagDelete(ctx context.Context, id uuid.UUID) error

// Rag Manager - Documents
RagV1DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType rmdocument.DocType, sourceURL string, storageFileID uuid.UUID) (*rmdocument.Document, error)
RagV1DocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error)
RagV1DocumentGets(ctx context.Context, pageToken string, pageSize uint64, filters map[rmdocument.Field]any) ([]*rmdocument.Document, error)
RagV1DocumentDelete(ctx context.Context, id uuid.UUID) error
```

Add imports to main.go:
```go
rmrag "monorepo/bin-rag-manager/models/rag"
rmdocument "monorepo/bin-rag-manager/models/document"
```

**Step 4: Run verification**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```
git add bin-common-handler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-common-handler: Add requesthandler RPC methods for rag CRUD (create, get, list, update, delete)
- bin-common-handler: Add requesthandler RPC methods for rag document CRUD (create, get, list, delete)"
```

---

### Task 3: Add OpenAPI schemas and paths

**Why:** bin-api-manager generates server code from the OpenAPI spec. We need schemas for rags/documents and path definitions for all endpoints.

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` — add schemas, path refs, tag
- Create: `bin-openapi-manager/openapi/paths/rags/main.yaml` — GET list + POST create
- Create: `bin-openapi-manager/openapi/paths/rags/id.yaml` — GET + PUT + DELETE
- Create: `bin-openapi-manager/openapi/paths/rag-documents/main.yaml` — GET list + POST create
- Create: `bin-openapi-manager/openapi/paths/rag-documents/id.yaml` — GET + DELETE

**Step 1: Add schemas to openapi.yaml**

Add to `components/schemas` section in `bin-openapi-manager/openapi/openapi.yaml`:

```yaml
    # Rag Manager schemas
    RagManagerRag:
      type: object
      properties:
        id:
          type: string
          format: uuid
          description: "The unique identifier of the rag. Returned from the `POST /rags` response."
          example: "550e8400-e29b-41d4-a716-446655440000"
        customer_id:
          type: string
          format: uuid
          description: "The customer ID that owns this rag."
          example: "7d4e8f2a-1b3c-4d5e-9f6a-8b7c6d5e4f3a"
        name:
          type: string
          description: "Human-readable name for the rag."
          example: "Customer Support KB"
        description:
          type: string
          description: "Description of what this rag contains."
          example: "Knowledge base for customer support conversations"
        tm_create:
          type: string
          format: date-time
          description: "Timestamp when the rag was created."
          example: "2026-03-18T10:30:00Z"
        tm_update:
          type: string
          format: date-time
          description: "Timestamp when the rag was last updated."
          example: "2026-03-18T10:30:00Z"

    RagManagerRagDocument:
      type: object
      properties:
        id:
          type: string
          format: uuid
          description: "The unique identifier of the document. Returned from the `POST /rag-documents` response."
          example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
        customer_id:
          type: string
          format: uuid
          description: "The customer ID that owns this document."
          example: "7d4e8f2a-1b3c-4d5e-9f6a-8b7c6d5e4f3a"
        rag_id:
          type: string
          format: uuid
          description: "The rag this document belongs to. Returned from the `POST /rags` response."
          example: "550e8400-e29b-41d4-a716-446655440000"
        name:
          type: string
          description: "Human-readable name for the document."
          example: "FAQ Document"
        doc_type:
          $ref: '#/components/schemas/RagManagerRagDocumentDocType'
        storage_file_id:
          type: string
          format: uuid
          description: "The storage file ID if doc_type is uploaded. Returned from the `POST /files` response."
          example: "b2c3d4e5-f6a7-8901-bcde-f12345678901"
        source_url:
          type: string
          description: "The source URL if doc_type is url."
          example: "https://example.com/docs/faq.html"
        status:
          $ref: '#/components/schemas/RagManagerRagDocumentStatus'
        status_message:
          type: string
          description: "Additional details about the current status."
          example: ""
        tm_create:
          type: string
          format: date-time
          description: "Timestamp when the document was created."
          example: "2026-03-18T11:00:00Z"
        tm_update:
          type: string
          format: date-time
          description: "Timestamp when the document was last updated."
          example: "2026-03-18T11:15:00Z"

    RagManagerRagDocumentDocType:
      type: string
      enum:
        - uploaded
        - url
        - platform
        - generated
      x-enum-varnames:
        - RagManagerRagDocumentDocTypeUploaded
        - RagManagerRagDocumentDocTypeUrl
        - RagManagerRagDocumentDocTypePlatform
        - RagManagerRagDocumentDocTypeGenerated
      example: "uploaded"

    RagManagerRagDocumentStatus:
      type: string
      enum:
        - pending
        - processing
        - ready
        - error
      x-enum-varnames:
        - RagManagerRagDocumentStatusPending
        - RagManagerRagDocumentStatusProcessing
        - RagManagerRagDocumentStatusReady
        - RagManagerRagDocumentStatusError
      example: "ready"
```

**Step 2: Add path references to openapi.yaml**

Add to `paths:` section in `bin-openapi-manager/openapi/openapi.yaml`:

```yaml
  /rags:
    $ref: './paths/rags/main.yaml'
  /rags/{id}:
    $ref: './paths/rags/id.yaml'
  /rag-documents:
    $ref: './paths/rag-documents/main.yaml'
  /rag-documents/{id}:
    $ref: './paths/rag-documents/id.yaml'
```

Add to `tags:` section:

```yaml
  - name: RAG
    description: RAG (Retrieval-Augmented Generation) knowledge base management
```

**Step 3: Create rags/main.yaml**

Create `bin-openapi-manager/openapi/paths/rags/main.yaml`:

```yaml
get:
  summary: Get a list of rags
  description: Retrieves a paginated list of RAG knowledge bases for the authenticated customer.
  tags:
    - RAG
  parameters:
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
  responses:
    '200':
      description: A list of rags.
      content:
        application/json:
          schema:
            allOf:
              - $ref: '#/components/schemas/CommonPagination'
              - type: object
                properties:
                  result:
                    type: array
                    items:
                      $ref: '#/components/schemas/RagManagerRag'

post:
  summary: Create a new rag
  description: Creates a new RAG knowledge base.
  tags:
    - RAG
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - name
          properties:
            name:
              type: string
              description: "Human-readable name for the rag."
              example: "Customer Support KB"
            description:
              type: string
              description: "Description of what this rag contains."
              example: "Knowledge base for customer support conversations"
  responses:
    '200':
      description: Successfully created rag.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRag'
```

**Step 4: Create rags/id.yaml**

Create `bin-openapi-manager/openapi/paths/rags/id.yaml`:

```yaml
get:
  summary: Get rag details
  description: Retrieves detailed information about a specific RAG knowledge base.
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
  responses:
    '200':
      description: Rag details.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRag'

put:
  summary: Update a rag
  description: Updates the name and/or description of an existing RAG knowledge base.
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
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          properties:
            name:
              type: string
              description: "Updated name for the rag."
              example: "Updated Support KB"
            description:
              type: string
              description: "Updated description."
              example: "Updated knowledge base description"
  responses:
    '200':
      description: Updated rag.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRag'

delete:
  summary: Delete a rag
  description: Deletes a RAG knowledge base and all associated documents and chunks.
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
  responses:
    '200':
      description: Rag deleted.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRag'
```

**Step 5: Create rag-documents/main.yaml**

Create `bin-openapi-manager/openapi/paths/rag-documents/main.yaml`:

```yaml
get:
  summary: Get a list of rag documents
  description: Retrieves a paginated list of documents in a RAG knowledge base.
  tags:
    - RAG
  parameters:
    - $ref: '#/components/parameters/PageSize'
    - $ref: '#/components/parameters/PageToken'
    - name: rag_id
      in: query
      required: false
      schema:
        type: string
        format: uuid
      description: "Filter documents by rag ID. Returned from the `POST /rags` response."
  responses:
    '200':
      description: A list of rag documents.
      content:
        application/json:
          schema:
            allOf:
              - $ref: '#/components/schemas/CommonPagination'
              - type: object
                properties:
                  result:
                    type: array
                    items:
                      $ref: '#/components/schemas/RagManagerRagDocument'

post:
  summary: Create a new rag document
  description: Creates a new document in a RAG knowledge base. Ingestion is asynchronous — the document status starts as `pending` and progresses to `ready` or `error`.
  tags:
    - RAG
  requestBody:
    required: true
    content:
      application/json:
        schema:
          type: object
          required:
            - rag_id
            - name
            - doc_type
          properties:
            rag_id:
              type: string
              format: uuid
              description: "The rag this document belongs to. Returned from the `POST /rags` response."
              example: "550e8400-e29b-41d4-a716-446655440000"
            name:
              type: string
              description: "Human-readable name for the document."
              example: "FAQ Document"
            doc_type:
              $ref: '#/components/schemas/RagManagerRagDocumentDocType'
            source_url:
              type: string
              description: "The source URL if doc_type is url."
              example: "https://example.com/docs/faq.html"
            storage_file_id:
              type: string
              format: uuid
              description: "The storage file ID if doc_type is uploaded. Returned from the `POST /files` response."
              example: "b2c3d4e5-f6a7-8901-bcde-f12345678901"
  responses:
    '200':
      description: Successfully created rag document.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRagDocument'
```

**Step 6: Create rag-documents/id.yaml**

Create `bin-openapi-manager/openapi/paths/rag-documents/id.yaml`:

```yaml
get:
  summary: Get rag document details
  description: Retrieves detailed information about a specific document in a RAG knowledge base.
  tags:
    - RAG
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
      description: "The unique identifier of the document. Returned from the `POST /rag-documents` response."
  responses:
    '200':
      description: Rag document details.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRagDocument'

delete:
  summary: Delete a rag document
  description: Deletes a document from a RAG knowledge base and all associated chunks.
  tags:
    - RAG
  parameters:
    - name: id
      in: path
      required: true
      schema:
        type: string
        format: uuid
      description: "The unique identifier of the document. Returned from the `POST /rag-documents` response."
  responses:
    '200':
      description: Rag document deleted.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/RagManagerRagDocument'
```

**Step 7: Regenerate OpenAPI models**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./...
```

Verify the generated `gens/models/gen.go` contains `RagManagerRag`, `RagManagerRagDocument`, etc.

**Step 8: Commit**

```
git add bin-openapi-manager/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-openapi-manager: Add RagManagerRag and RagManagerRagDocument schemas
- bin-openapi-manager: Add RagManagerRagDocumentDocType and RagManagerRagDocumentStatus enums
- bin-openapi-manager: Add OpenAPI paths for /rags (GET list, POST create, GET id, PUT id, DELETE id)
- bin-openapi-manager: Add OpenAPI paths for /rag-documents (GET list, POST create, GET id, DELETE id)"
```

---

### Task 4: Add servicehandler methods in bin-api-manager

**Why:** ServiceHandler methods handle authorization (PermissionCustomerAdmin), call requesthandler, and convert to WebhookMessage.

**Files:**
- Create: `bin-api-manager/pkg/servicehandler/rag.go` — rag CRUD servicehandler methods
- Create: `bin-api-manager/pkg/servicehandler/rag_document.go` — rag-document CRUD servicehandler methods
- Modify: `bin-api-manager/pkg/servicehandler/main.go` — add methods to interface

**Step 1: Create rag.go servicehandler**

Create `bin-api-manager/pkg/servicehandler/rag.go`:

```go
package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmrag "monorepo/bin-rag-manager/models/rag"
)

// ragGet is the private helper — fetches rag without permission check.
func (h *serviceHandler) ragGet(ctx context.Context, id uuid.UUID) (*rmrag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ragGet",
		"rag_id": id,
	})

	res, err := h.reqHandler.RagV1RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the rag info. err: %v", err)
		return nil, err
	}
	log.WithField("rag", res).Debug("Received result.")

	return res, nil
}

func (h *serviceHandler) RagCreate(ctx context.Context, a *amagent.Agent, name, description string) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": a.CustomerID,
		"name":        name,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	log.Debug("Creating a new rag.")
	tmp, err := h.reqHandler.RagV1RagCreate(ctx, a.CustomerID, name, description)
	if err != nil {
		log.Errorf("Could not create a new rag. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagGet",
		"customer_id": a.CustomerID,
		"rag_id":      id,
	})
	log.Debug("Getting a rag.")

	tmp, err := h.ragGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagGets",
		"customer_id": a.CustomerID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting rags.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	filters := map[rmrag.Field]any{
		rmrag.FieldCustomerID: a.CustomerID,
	}
	rags, err := h.reqHandler.RagV1RagGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get rags info. err: %v", err)
		return nil, fmt.Errorf("could not find rags info. err: %v", err)
	}

	res := []*rmrag.WebhookMessage{}
	for _, r := range rags {
		tmp := r.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

func (h *serviceHandler) RagUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagUpdate",
		"customer_id": a.CustomerID,
		"rag_id":      id,
	})
	log.Debug("Updating a rag.")

	tmp, err := h.ragGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	updated, err := h.reqHandler.RagV1RagUpdate(ctx, id, fields)
	if err != nil {
		log.Errorf("Could not update rag. err: %v", err)
		return nil, err
	}

	res := updated.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmrag.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDelete",
		"customer_id": a.CustomerID,
		"rag_id":      id,
	})
	log.Debug("Deleting a rag.")

	tmp, err := h.ragGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if err := h.reqHandler.RagV1RagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

**Step 2: Create rag_document.go servicehandler**

Create `bin-api-manager/pkg/servicehandler/rag_document.go`:

```go
package servicehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmdocument "monorepo/bin-rag-manager/models/document"
	rmrag "monorepo/bin-rag-manager/models/rag"
)

// ragDocumentGet is the private helper — fetches document without permission check.
func (h *serviceHandler) ragDocumentGet(ctx context.Context, id uuid.UUID) (*rmdocument.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ragDocumentGet",
		"document_id": id,
	})

	res, err := h.reqHandler.RagV1DocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the rag document info. err: %v", err)
		return nil, err
	}
	log.WithField("document", res).Debug("Received result.")

	return res, nil
}

func (h *serviceHandler) RagDocumentCreate(
	ctx context.Context,
	a *amagent.Agent,
	ragID uuid.UUID,
	name string,
	docType rmdocument.DocType,
	sourceURL string,
	storageFileID uuid.UUID,
) (*rmdocument.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDocumentCreate",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
		"name":        name,
	})

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	// verify rag exists and belongs to this customer
	r, err := h.ragGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get rag info. err: %v", err)
		return nil, fmt.Errorf("could not find rag info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, r.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	log.Debug("Creating a new rag document.")
	tmp, err := h.reqHandler.RagV1DocumentCreate(ctx, a.CustomerID, ragID, name, docType, sourceURL, storageFileID)
	if err != nil {
		log.Errorf("Could not create a new rag document. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagDocumentGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdocument.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDocumentGet",
		"customer_id": a.CustomerID,
		"document_id": id,
	})
	log.Debug("Getting a rag document.")

	tmp, err := h.ragDocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag document info. err: %v", err)
		return nil, fmt.Errorf("could not find rag document info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}

func (h *serviceHandler) RagDocumentGets(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, size uint64, token string) ([]*rmdocument.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDocumentGets",
		"customer_id": a.CustomerID,
		"rag_id":      ragID,
		"size":        size,
		"token":       token,
	})
	log.Debug("Getting rag documents.")

	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	filters := map[rmdocument.Field]any{
		rmdocument.FieldCustomerID: a.CustomerID,
	}
	if ragID != uuid.Nil {
		filters[rmdocument.FieldRagID] = ragID
	}

	docs, err := h.reqHandler.RagV1DocumentGets(ctx, token, size, filters)
	if err != nil {
		log.Errorf("Could not get rag documents info. err: %v", err)
		return nil, fmt.Errorf("could not find rag documents info. err: %v", err)
	}

	res := []*rmdocument.WebhookMessage{}
	for _, d := range docs {
		tmp := d.ConvertWebhookMessage()
		res = append(res, tmp)
	}

	return res, nil
}

func (h *serviceHandler) RagDocumentDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdocument.WebhookMessage, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagDocumentDelete",
		"customer_id": a.CustomerID,
		"document_id": id,
	})
	log.Debug("Deleting a rag document.")

	tmp, err := h.ragDocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag document info. err: %v", err)
		return nil, fmt.Errorf("could not find rag document info. err: %v", err)
	}

	if !h.hasPermission(ctx, a, tmp.CustomerID, amagent.PermissionCustomerAdmin) {
		return nil, fmt.Errorf("user has no permission")
	}

	if err := h.reqHandler.RagV1DocumentDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag document. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertWebhookMessage()
	return res, nil
}
```

**Step 3: Update ServiceHandler interface in main.go**

Add to `bin-api-manager/pkg/servicehandler/main.go`:

```go
// RAG
RagCreate(ctx context.Context, a *amagent.Agent, name, description string) (*rmrag.WebhookMessage, error)
RagGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmrag.WebhookMessage, error)
RagGets(ctx context.Context, a *amagent.Agent, size uint64, token string) ([]*rmrag.WebhookMessage, error)
RagUpdate(ctx context.Context, a *amagent.Agent, id uuid.UUID, fields map[rmrag.Field]any) (*rmrag.WebhookMessage, error)
RagDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmrag.WebhookMessage, error)

// RAG Documents
RagDocumentCreate(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, name string, docType rmdocument.DocType, sourceURL string, storageFileID uuid.UUID) (*rmdocument.WebhookMessage, error)
RagDocumentGet(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdocument.WebhookMessage, error)
RagDocumentGets(ctx context.Context, a *amagent.Agent, ragID uuid.UUID, size uint64, token string) ([]*rmdocument.WebhookMessage, error)
RagDocumentDelete(ctx context.Context, a *amagent.Agent, id uuid.UUID) (*rmdocument.WebhookMessage, error)
```

Add imports:
```go
rmrag "monorepo/bin-rag-manager/models/rag"
rmdocument "monorepo/bin-rag-manager/models/document"
```

**Step 4: Verify it compiles**

```bash
cd bin-api-manager && go build ./...
```

**Step 5: Commit**

```
git add bin-api-manager/pkg/servicehandler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-api-manager: Add rag servicehandler (create, get, list, update, delete) with PermissionCustomerAdmin
- bin-api-manager: Add rag-document servicehandler (create, get, list, delete) with PermissionCustomerAdmin"
```

---

### Task 5: Add server HTTP handlers in bin-api-manager

**Why:** Server handlers parse HTTP requests, call servicehandler, and return JSON responses. They are generated from OpenAPI spec and must match the generated function signatures.

**Files:**
- Create: `bin-api-manager/server/rags.go` — HTTP handlers for rags
- Create: `bin-api-manager/server/rag_documents.go` — HTTP handlers for rag-documents

**Important:** First regenerate api-manager code from updated OpenAPI spec:

```bash
cd bin-api-manager && go generate ./...
```

Check `bin-api-manager/gens/openapi_server/gen.go` for the generated function signatures (`GetRags`, `PostRags`, `GetRagsId`, `PutRagsId`, `DeleteRagsId`, `GetRagDocuments`, `PostRagDocuments`, `GetRagDocumentsId`, `DeleteRagDocumentsId`). Match these signatures exactly.

**Step 1: Create rags.go**

Create `bin-api-manager/server/rags.go`. Follow the campaign pattern: extract agent from context, parse request, call servicehandler, return JSON.

```go
package server

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	openapi_server "monorepo/bin-api-manager/gens/openapi_server"
	rmrag "monorepo/bin-rag-manager/models/rag"

	"github.com/gin-gonic/gin"
)

func (h *server) PostRags(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRags",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	var req openapi_server.PostRagsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	res, err := h.serviceHandler.RagCreate(c.Request.Context(), &a, name, description)
	if err != nil {
		log.Errorf("Could not create a rag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetRags(c *gin.Context, params openapi_server.GetRagsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRags",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.RagGets(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get rags. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetRagsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRagsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a rag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutRagsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutRagsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutRagsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	fields := map[rmrag.Field]any{}
	if req.Name != nil {
		fields[rmrag.FieldName] = *req.Name
	}
	if req.Description != nil {
		fields[rmrag.FieldDescription] = *req.Description
	}

	res, err := h.serviceHandler.RagUpdate(c.Request.Context(), &a, target, fields)
	if err != nil {
		log.Errorf("Could not update a rag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteRagsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRagsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete a rag. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

**Step 2: Create rag_documents.go**

Create `bin-api-manager/server/rag_documents.go`:

```go
package server

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	openapi_server "monorepo/bin-api-manager/gens/openapi_server"
	rmdocument "monorepo/bin-rag-manager/models/document"

	"github.com/gin-gonic/gin"
)

func (h *server) PostRagDocuments(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostRagDocuments",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	var req openapi_server.PostRagDocumentsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	ragID := uuid.FromStringOrNil("")
	if req.RagId != nil {
		ragID = uuid.FromStringOrNil(*req.RagId)
	}
	if ragID == uuid.Nil {
		log.Error("rag_id is required.")
		c.AbortWithStatus(400)
		return
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	docType := rmdocument.DocType("")
	if req.DocType != nil {
		docType = rmdocument.DocType(*req.DocType)
	}

	sourceURL := ""
	if req.SourceUrl != nil {
		sourceURL = *req.SourceUrl
	}

	storageFileID := uuid.Nil
	if req.StorageFileId != nil {
		storageFileID = uuid.FromStringOrNil(*req.StorageFileId)
	}

	res, err := h.serviceHandler.RagDocumentCreate(c.Request.Context(), &a, ragID, name, docType, sourceURL, storageFileID)
	if err != nil {
		log.Errorf("Could not create a rag document. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetRagDocuments(c *gin.Context, params openapi_server.GetRagDocumentsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRagDocuments",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	// Extract rag_id from query params if provided
	ragID := uuid.Nil
	if params.RagId != nil {
		ragID = uuid.FromStringOrNil(*params.RagId)
	}

	tmps, err := h.serviceHandler.RagDocumentGets(c.Request.Context(), &a, ragID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get rag documents. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) GetRagDocumentsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetRagDocumentsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagDocumentGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a rag document. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteRagDocumentsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteRagDocumentsId",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.RagDocumentDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete a rag document. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}
```

**Step 3: Regenerate and verify bin-api-manager**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go build ./...
```

**Step 4: Commit**

```
git add bin-api-manager/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-api-manager: Add HTTP handlers for /rags (GET list, POST create, GET id, PUT id, DELETE id)
- bin-api-manager: Add HTTP handlers for /rag-documents (GET list, POST create, GET id, DELETE id)
- bin-api-manager: Add servicehandler interface methods for rag and rag-document CRUD"
```

---

### Task 6: Full verification across all services

**Step 1: Verify bin-rag-manager**

```bash
cd bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Verify bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Verify bin-openapi-manager**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && golangci-lint run -v --timeout 5m
```

**Step 4: Verify bin-api-manager**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

**Step 6: Final commit if verification produced changes**

```
git add -A
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-rag-manager: Verification updates (go.mod, go.sum, generated mocks)
- bin-common-handler: Verification updates (go.mod, go.sum, generated mocks)
- bin-openapi-manager: Verification updates (go.mod, go.sum, generated models)
- bin-api-manager: Verification updates (go.mod, go.sum, generated server code)"
```

**Step 7: Push and update PR**

```bash
git push origin NOJIRA-Rag-manager-phase2-api-endpoints
```
