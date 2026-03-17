# Rag Manager Phase 2: API Endpoints Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement all 9 RabbitMQ RPC endpoints for bin-rag-manager (rag CRUD, document CRUD, query) so pipecat-manager can use the RAG pipeline.

**Architecture:** Listenhandler receives RPC requests, routes via regex to handler methods that parse request data and call raghandler. Raghandler orchestrates dbhandler (PostgreSQL/pgvector) and embedder (Vertex AI). List endpoints use filter-based pagination matching the monorepo pattern.

**Tech Stack:** Go, PostgreSQL with pgvector, Google Vertex AI (text-embedding-004), RabbitMQ RPC, squirrel query builder

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Rag-manager-phase2-api-endpoints`

**Design doc:** `docs/plans/2026-03-18-rag-manager-phase2-api-endpoints-design.md`

---

### Task 1: Update models — FieldStruct filters and remove Answer

**Files:**
- Create: `bin-rag-manager/models/rag/filters.go`
- Create: `bin-rag-manager/models/document/filters.go`
- Modify: `bin-rag-manager/models/query/main.go:23-26`

**Step 1: Create rag FieldStruct**

Create `bin-rag-manager/models/rag/filters.go`:

```go
package rag

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Rag queries.
// Used by utilhandler.ConvertFilters to validate and type-convert filter values.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Deleted    bool      `filter:"deleted"`
}
```

**Step 2: Create document FieldStruct**

Create `bin-rag-manager/models/document/filters.go`:

```go
package document

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Document queries.
// Used by utilhandler.ConvertFilters to validate and type-convert filter values.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	RagID      uuid.UUID `filter:"rag_id"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}
```

**Step 3: Remove Answer from query.Response**

Edit `bin-rag-manager/models/query/main.go` — remove the `Answer` field:

```go
// Response represents a RAG query response — sources only.
// Answer generation is handled by the caller (e.g., pipecat-manager).
type Response struct {
	Sources []Source `json:"sources"`
}
```

**Step 4: Commit**

```
git add bin-rag-manager/models/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-rag-manager: Add FieldStruct filters for rag and document models
- bin-rag-manager: Remove Answer field from query.Response"
```

---

### Task 2: Update DBHandler — replace specific list methods with filter-based list

**Files:**
- Modify: `bin-rag-manager/pkg/dbhandler/main.go:27-28,34-35`
- Modify: `bin-rag-manager/pkg/dbhandler/rag.go:141-166`
- Modify: `bin-rag-manager/pkg/dbhandler/document.go:194-246`

**Step 1: Update DBHandler interface**

In `bin-rag-manager/pkg/dbhandler/main.go`, replace the specific Gets methods with generic List methods:

Replace:
```go
RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error)
```
With:
```go
RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error)
```

Replace:
```go
DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error)
DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error)
```
With:
```go
DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error)
```

**Step 2: Implement RagList in rag.go**

Replace `RagGetsByCustomerID` with `RagList`:

```go
// RagList retrieves rags matching the given filters with cursor-based pagination.
// Pagination uses tm_create as cursor (token). Filters are applied as WHERE clauses.
// The "deleted" filter controls tm_delete: false = IS NULL, true = IS NOT NULL.
func (h *handler) RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}
	if size == 0 {
		size = 100
	}

	q := psql.
		Select(ragColumns()...).
		From(tableRagRags).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create DESC").
		Limit(size)

	for k, v := range filters {
		key := string(k)
		switch key {
		case "deleted":
			deleted, ok := v.(bool)
			if ok && !deleted {
				q = q.Where("tm_delete IS NULL")
			} else if ok && deleted {
				q = q.Where("tm_delete IS NOT NULL")
			}
		default:
			q = q.Where(sq.Eq{key: v})
		}
	}

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build rag list query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute rag list query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return ragScanRows(rows)
}
```

**Step 3: Implement DocumentList in document.go**

Replace `DocumentGetsByRagID` and `DocumentGetsByCustomerID` with a single `DocumentList`:

```go
// DocumentList retrieves documents matching the given filters with cursor-based pagination.
func (h *handler) DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error) {
	if token == "" {
		token = h.utilHandler.TimeGetCurTime()
	}
	if size == 0 {
		size = 100
	}

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Lt{"tm_create": token}).
		OrderBy("tm_create DESC").
		Limit(size)

	for k, v := range filters {
		key := string(k)
		switch key {
		case "deleted":
			deleted, ok := v.(bool)
			if ok && !deleted {
				q = q.Where("tm_delete IS NULL")
			} else if ok && deleted {
				q = q.Where("tm_delete IS NOT NULL")
			}
		default:
			q = q.Where(sq.Eq{key: v})
		}
	}

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build document list query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not execute document list query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDocumentRows(rows)
}
```

**Step 4: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`

**Step 5: Commit**

```
git add bin-rag-manager/pkg/dbhandler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-rag-manager: Replace RagGetsByCustomerID with filter-based RagList
- bin-rag-manager: Replace DocumentGetsByRagID/CustomerID with filter-based DocumentList"
```

---

### Task 3: Implement RagHandler business logic

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/main.go:19-32` (update interface)
- Create: `bin-rag-manager/pkg/raghandler/rag.go`
- Create: `bin-rag-manager/pkg/raghandler/document.go`
- Create: `bin-rag-manager/pkg/raghandler/query.go`

**Step 1: Update RagHandler interface**

In `main.go`, update the interface to match new dbhandler signatures and add list methods with filters:

```go
type RagHandler interface {
	RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rag.Rag, error)
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error)
	RagDelete(ctx context.Context, id uuid.UUID) error

	DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType document.DocType, sourceURL string, storageFileID uuid.UUID) (*document.Document, error)
	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error)
	DocumentDelete(ctx context.Context, id uuid.UUID) error

	QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error)
}
```

Remove old stub methods from `main.go` (lines 50-98). Move implementations to separate files.

**Step 2: Implement rag.go — Rag CRUD**

Create `bin-rag-manager/pkg/raghandler/rag.go`:

```go
package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/rag"
)

func (h *ragHandler) RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "RagCreate",
		"customer_id": customerID,
		"name":        name,
	})

	id, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate UUID. err: %v", err)
		return nil, fmt.Errorf("could not generate rag id: %w", err)
	}

	r := &rag.Rag{
		ID:          id,
		CustomerID:  customerID,
		Name:        name,
		Description: description,
	}

	if errCreate := h.dbHandler.RagCreate(ctx, r); errCreate != nil {
		log.Errorf("Could not create rag. err: %v", errCreate)
		return nil, fmt.Errorf("could not create rag: %w", errCreate)
	}
	log.WithField("rag", r).Debugf("Created rag. rag_id: %s", r.ID)

	return r, nil
}

func (h *ragHandler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagGet",
		"id":   id,
	})

	r, err := h.dbHandler.RagGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get rag. err: %v", err)
		return nil, fmt.Errorf("could not get rag: %w", err)
	}
	log.WithField("rag", r).Debugf("Retrieved rag. rag_id: %s", r.ID)

	return r, nil
}

func (h *ragHandler) RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "RagList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	rags, err := h.dbHandler.RagList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list rags. err: %v", err)
		return nil, fmt.Errorf("could not list rags: %w", err)
	}
	log.Debugf("Listed rags. count: %d", len(rags))

	return rags, nil
}

func (h *ragHandler) RagDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "RagDelete",
		"id":   id,
	})

	// cascade: soft-delete chunks and documents first
	if err := h.dbHandler.ChunkSoftDeleteByRagID(ctx, id); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return fmt.Errorf("could not delete rag chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDeleteByRagID(ctx, id); err != nil {
		log.Errorf("Could not soft delete documents. err: %v", err)
		return fmt.Errorf("could not delete rag documents: %w", err)
	}

	if err := h.dbHandler.RagDelete(ctx, id); err != nil {
		log.Errorf("Could not delete rag. err: %v", err)
		return fmt.Errorf("could not delete rag: %w", err)
	}
	log.Debugf("Deleted rag. rag_id: %s", id)

	return nil
}
```

**Step 3: Implement document.go — Document CRUD with async ingestion**

Create `bin-rag-manager/pkg/raghandler/document.go`:

```go
package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/document"
)

func (h *ragHandler) DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType document.DocType, sourceURL string, storageFileID uuid.UUID) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DocumentCreate",
		"customer_id": customerID,
		"rag_id":      ragID,
		"name":        name,
		"doc_type":    docType,
	})

	id, err := uuid.NewV4()
	if err != nil {
		log.Errorf("Could not generate UUID. err: %v", err)
		return nil, fmt.Errorf("could not generate document id: %w", err)
	}

	d := &document.Document{
		ID:            id,
		CustomerID:    customerID,
		RagID:         ragID,
		Name:          name,
		DocType:       docType,
		SourceURL:     sourceURL,
		StorageFileID: storageFileID,
		Status:        document.StatusPending,
	}

	if errCreate := h.dbHandler.DocumentCreate(ctx, d); errCreate != nil {
		log.Errorf("Could not create document. err: %v", errCreate)
		return nil, fmt.Errorf("could not create document: %w", errCreate)
	}
	log.WithField("document", d).Debugf("Created document. document_id: %s", d.ID)

	// TODO: trigger async ingestion goroutine (Phase 2b — chunking + embedding pipeline)

	return d, nil
}

func (h *ragHandler) DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "DocumentGet",
		"id":   id,
	})

	d, err := h.dbHandler.DocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get document. err: %v", err)
		return nil, fmt.Errorf("could not get document: %w", err)
	}
	log.WithField("document", d).Debugf("Retrieved document. document_id: %s", d.ID)

	return d, nil
}

func (h *ragHandler) DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DocumentList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	docs, err := h.dbHandler.DocumentList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list documents. err: %v", err)
		return nil, fmt.Errorf("could not list documents: %w", err)
	}
	log.Debugf("Listed documents. count: %d", len(docs))

	return docs, nil
}

func (h *ragHandler) DocumentDelete(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "DocumentDelete",
		"id":   id,
	})

	// cascade: soft-delete chunks first
	if err := h.dbHandler.ChunkSoftDeleteByDocumentID(ctx, id); err != nil {
		log.Errorf("Could not soft delete chunks. err: %v", err)
		return fmt.Errorf("could not delete document chunks: %w", err)
	}

	if err := h.dbHandler.DocumentDelete(ctx, id); err != nil {
		log.Errorf("Could not delete document. err: %v", err)
		return fmt.Errorf("could not delete document: %w", err)
	}
	log.Debugf("Deleted document. document_id: %s", id)

	return nil
}
```

**Step 4: Implement query.go — Vector search**

Create `bin-rag-manager/pkg/raghandler/query.go`:

```go
package raghandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/models/query"
)

func (h *ragHandler) QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "QueryRag",
		"rag_id": ragID,
		"top_k":  topK,
	})

	if topK <= 0 {
		topK = 5
	}

	// embed the query text
	embedding, err := h.embedder.EmbedText(ctx, queryText)
	if err != nil {
		log.Errorf("Could not embed query. err: %v", err)
		return nil, fmt.Errorf("could not embed query: %w", err)
	}

	// vector similarity search
	chunks, scores, err := h.dbHandler.ChunkSearchByRagID(ctx, ragID, embedding, topK)
	if err != nil {
		log.Errorf("Could not search chunks. err: %v", err)
		return nil, fmt.Errorf("could not search chunks: %w", err)
	}
	log.Debugf("Found %d matching chunks for rag_id: %s", len(chunks), ragID)

	// build sources from chunks + scores
	sources := make([]query.Source, len(chunks))
	for i, c := range chunks {
		// look up document name
		docName := ""
		doc, errDoc := h.dbHandler.DocumentGet(ctx, c.DocumentID)
		if errDoc == nil {
			docName = doc.Name
		}

		sources[i] = query.Source{
			DocumentID:     c.DocumentID,
			DocumentName:   docName,
			SectionTitle:   c.SectionTitle,
			RelevanceScore: scores[i],
		}
	}

	return &query.Response{
		Sources: sources,
	}, nil
}
```

**Step 5: Clean up main.go stubs**

Remove all stub method implementations from `main.go` (lines 50-98). Keep only the interface, struct, and constructor.

**Step 6: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`

**Step 7: Commit**

```
git add bin-rag-manager/pkg/raghandler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-rag-manager: Implement RagHandler rag CRUD (create, get, list, delete with cascade)
- bin-rag-manager: Implement RagHandler document CRUD (create with async placeholder, get, list, delete with cascade)
- bin-rag-manager: Implement RagHandler query (embed query, vector search, return sources)"
```

---

### Task 4: Implement ListenHandler routes and handler methods

**Files:**
- Modify: `bin-rag-manager/pkg/listenhandler/main.go:5-18,69-92`
- Create: `bin-rag-manager/pkg/listenhandler/v1_rags.go`
- Create: `bin-rag-manager/pkg/listenhandler/v1_documents.go`
- Create: `bin-rag-manager/pkg/listenhandler/v1_query.go`

**Step 1: Add regex patterns and route matching to main.go**

Add imports and regex patterns before the `ListenHandler` interface. Update the `processRequest` switch:

```go
package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-rag-manager/pkg/raghandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// rag routes
	regV1Rags   = regexp.MustCompile(`^/v1/rags(\?.*)?$`)
	regV1RagsID = regexp.MustCompile(`^/v1/rags/` + regUUID + `(\?.*)?$`)

	// document routes
	regV1Documents   = regexp.MustCompile(`^/v1/documents(\?.*)?$`)
	regV1DocumentsID = regexp.MustCompile(`^/v1/documents/` + regUUID + `(\?.*)?$`)

	// query route
	regV1Query = regexp.MustCompile(`^/v1/query$`)
)
```

Update `processRequest` switch:

```go
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "processRequest",
		"uri":    m.URI,
		"method": m.Method,
	})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	ctx := context.Background()
	start := time.Now()
	var requestType string
	var response *sock.Response
	var err error

	switch {
	// rag routes — ID routes before collection routes
	case regV1RagsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RagsIDGet(ctx, m)
		requestType = "/v1/rags/<rag-id>"

	case regV1RagsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1RagsIDDelete(ctx, m)
		requestType = "/v1/rags/<rag-id>"

	case regV1Rags.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RagsPost(ctx, m)
		requestType = "/v1/rags"

	case regV1Rags.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RagsGet(ctx, m)
		requestType = "/v1/rags"

	// document routes — ID routes before collection routes
	case regV1DocumentsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DocumentsIDGet(ctx, m)
		requestType = "/v1/documents/<document-id>"

	case regV1DocumentsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1DocumentsIDDelete(ctx, m)
		requestType = "/v1/documents/<document-id>"

	case regV1Documents.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1DocumentsPost(ctx, m)
		requestType = "/v1/documents"

	case regV1Documents.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DocumentsGet(ctx, m)
		requestType = "/v1/documents"

	// query route
	case regV1Query.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1QueryPost(ctx, m)
		requestType = "/v1/query"

	default:
		log.Errorf("Could not find the handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		requestType = "notfound"
	}

	if err != nil {
		log.Errorf("Could not process request. err: %v", err)
		response = simpleResponse(500)
	}

	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	return response, nil
}
```

**Step 2: Create v1_rags.go**

Create `bin-rag-manager/pkg/listenhandler/v1_rags.go` with handlers for rag CRUD. Follow the monorepo pattern: parse URI/query params, parse filters from body, call raghandler, marshal response.

Key patterns:
- `url.Parse(m.URI)` for query params
- `utilhandler.ParseFiltersFromRequestBody(m.Data)` + `utilhandler.ConvertFilters` for list filters
- `strings.Split(m.URI, "/")` to extract UUID from path (index 3 for `/v1/rags/<id>`)
- `uuid.FromString(uriItems[3])` to parse UUID

**Step 3: Create v1_documents.go**

Same pattern as v1_rags.go but for document CRUD. Document creation parses `rag_id`, `name`, `doc_type`, `source_url`, `storage_file_id` from request body JSON.

**Step 4: Create v1_query.go**

Parse `rag_id`, `query`, `top_k` from request body. Call `h.ragHandler.QueryRag(ctx, ragID, queryText, topK)`. Return JSON response.

**Step 5: Verify it compiles**

Run: `cd bin-rag-manager && go build ./...`

**Step 6: Commit**

```
git add bin-rag-manager/pkg/listenhandler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-rag-manager: Add regex route patterns for all 9 endpoints
- bin-rag-manager: Implement rag CRUD handlers (create, get, list, delete)
- bin-rag-manager: Implement document CRUD handlers (create, get, list, delete)
- bin-rag-manager: Implement query handler (POST /v1/query)"
```

---

### Task 5: Update bin-common-handler caller

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/rag_rags.go`
- Modify: `bin-common-handler/pkg/requesthandler/main.go:1314`

**Step 1: Update RagV1RagQuery signature and implementation**

In `main.go` line 1314, change the interface:

```go
RagV1RagQuery(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*rmquery.Response, error)
```

In `rag_rags.go`, update the method:

```go
func (r *requestHandler) RagV1RagQuery(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*rmquery.Response, error) {
	uri := "/v1/query"

	req := struct {
		RagID uuid.UUID `json:"rag_id"`
		Query string    `json:"query"`
		TopK  int       `json:"top_k,omitempty"`
	}{
		RagID: ragID,
		Query: queryText,
		TopK:  topK,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal request")
	}

	tmp, err := r.sendRequestRag(ctx, uri, sock.RequestMethodPost, "rag/query", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res rmquery.Response
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
```

Note: add `"github.com/gofrs/uuid"` to imports in rag_rags.go. The `main.go` import for `rmquery` stays the same.

**Step 2: Run verification for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```
git add bin-common-handler/
git commit -m "NOJIRA-Rag-manager-phase2-api-endpoints

- bin-common-handler: Update RagV1RagQuery to accept ragID, change URI to /v1/query, remove docTypes"
```

---

### Task 6: Full verification and final commit

**Step 1: Run full verification for bin-rag-manager**

```bash
cd bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Run full verification for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Check for conflicts with main**

```bash
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

**Step 4: Push and create PR**

```bash
git push -u origin NOJIRA-Rag-manager-phase2-api-endpoints
gh pr create --title "NOJIRA-Rag-manager-phase2-api-endpoints" --body "..."
```
