# Reject Duplicate RAG Sources — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reject requests that add duplicate sources (same `storage_file_id` or `source_url`) to the same RAG.

**Architecture:** Add intra-request dedup + DB lookup in `createDocumentsForSources()` before creating any documents. Change `createDocumentsForSources()` from void to returning an error, propagating it through `RagCreate()` and `RagAddSources()`. Add a new `DocumentGetsByRagIDAndSources()` DB method for the lookup.

**Tech Stack:** Go, squirrel query builder, gomock, PostgreSQL

---

## Design

**Problem:** The rag-manager allows adding the same `storage_file_id` or `source_url` to the same RAG multiple times, creating duplicate chunks and embeddings.

**Scope:**
- Duplicate detection scoped to the same RAG (same source can exist in different RAGs).
- Only active (non-deleted) documents checked — deleted sources can be re-added.
- If any duplicate found, entire request is rejected.

**Detection levels:**
1. Intra-request: reject if `storage_file_ids[]` or `source_urls[]` contain duplicates within the request itself.
2. DB-level: query active documents in the same RAG matching any of the requested sources.

**Known limitation:** Race condition between concurrent requests accepted for now.

---

### Task 1: Add `DocumentGetsByRagIDAndSources` to DBHandler

**Files:**
- Modify: `bin-rag-manager/pkg/dbhandler/document.go` (append new method)
- Modify: `bin-rag-manager/pkg/dbhandler/main.go:44` (add to interface)

**Step 1: Add the method to the `DBHandler` interface**

In `pkg/dbhandler/main.go`, add to the `DBHandler` interface after line 44 (`DocumentGetsByRagIDs`):

```go
DocumentGetsByRagIDAndSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) ([]*document.Document, error)
```

**Step 2: Implement the method**

In `pkg/dbhandler/document.go`, append:

```go
// DocumentGetsByRagIDAndSources returns active documents in a RAG that match any of the given sources.
// Used for duplicate detection before adding new sources.
func (h *handler) DocumentGetsByRagIDAndSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) ([]*document.Document, error) {
	if len(storageFileIDs) == 0 && len(sourceURLs) == 0 {
		return []*document.Document{}, nil
	}

	// Build OR conditions for each source type
	var orConditions []sq.Sqlizer

	if len(storageFileIDs) > 0 {
		orConditions = append(orConditions, sq.Eq{"storage_file_id": storageFileIDs})
	}
	if len(sourceURLs) > 0 {
		orConditions = append(orConditions, sq.Eq{"source_url": sourceURLs})
	}

	q := psql.
		Select(documentColumns()...).
		From(tableDocuments).
		Where(sq.Eq{"rag_id": ragID}).
		Where("tm_delete IS NULL").
		Where(sq.Or(orConditions))

	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("could not build duplicate check query: %w", err)
	}

	rows, err := h.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query duplicate sources: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanDocumentRows(rows)
}
```

**Step 3: Regenerate mocks**

Run: `cd bin-rag-manager && go generate ./...`

**Step 4: Run tests to verify nothing is broken**

Run: `cd bin-rag-manager && go test ./...`
Expected: All existing tests pass.

---

### Task 2: Add duplicate detection in `createDocumentsForSources`

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/rag.go:253-275` (`createDocumentsForSources` method)
- Modify: `bin-rag-manager/pkg/raghandler/rag.go:41` (call site in `RagCreate`)
- Modify: `bin-rag-manager/pkg/raghandler/rag.go:176` (call site in `RagAddSources`)

**Step 1: Change `createDocumentsForSources` signature to return an error and add duplicate detection**

Replace the existing `createDocumentsForSources` method (lines 251-275) with:

```go
// createDocumentsForSources creates documents for each file ID and URL, then triggers ingestion.
// Returns an error if duplicate sources are detected (within the request or already in the RAG).
// Uses request ctx for DB writes; ingestion goroutines use context.Background().
func (h *ragHandler) createDocumentsForSources(ctx context.Context, customerID, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":   "createDocumentsForSources",
		"rag_id": ragID,
	})

	// Intra-request duplicate check for storage file IDs
	seenFileIDs := make(map[uuid.UUID]bool, len(storageFileIDs))
	for _, fileID := range storageFileIDs {
		if seenFileIDs[fileID] {
			return fmt.Errorf("duplicate source(s) in request: storage_file_id %s", fileID)
		}
		seenFileIDs[fileID] = true
	}

	// Intra-request duplicate check for source URLs
	seenURLs := make(map[string]bool, len(sourceURLs))
	for _, u := range sourceURLs {
		if seenURLs[u] {
			return fmt.Errorf("duplicate source(s) in request: source_url %s", u)
		}
		seenURLs[u] = true
	}

	// DB-level duplicate check against existing active documents in this RAG
	existingDocs, err := h.dbHandler.DocumentGetsByRagIDAndSources(ctx, ragID, storageFileIDs, sourceURLs)
	if err != nil {
		log.Errorf("Could not check for duplicate sources. err: %v", err)
		return fmt.Errorf("could not check for duplicate sources: %w", err)
	}
	if len(existingDocs) > 0 {
		dupNames := make([]string, 0, len(existingDocs))
		for _, d := range existingDocs {
			if d.StorageFileID != uuid.Nil {
				dupNames = append(dupNames, "storage_file_id "+d.StorageFileID.String())
			} else if d.SourceURL != "" {
				dupNames = append(dupNames, "source_url "+d.SourceURL)
			}
		}
		return fmt.Errorf("source(s) already exist in this rag: %s", strings.Join(dupNames, ", "))
	}

	// Create documents and trigger ingestion
	for _, fileID := range storageFileIDs {
		doc, err := h.documentCreateInternal(ctx, customerID, ragID, fileID, "")
		if err != nil {
			log.Errorf("Could not create document for file_id %s: %v", fileID, err)
			continue
		}
		go h.documentIngest(doc)
	}

	for _, u := range sourceURLs {
		doc, err := h.documentCreateInternal(ctx, customerID, ragID, uuid.Nil, u)
		if err != nil {
			log.Errorf("Could not create document for url %s: %v", u, err)
			continue
		}
		go h.documentIngest(doc)
	}

	return nil
}
```

**Step 2: Add `"strings"` import to `rag.go`**

Add `"strings"` to the import block in `rag.go`.

**Step 3: Update `RagCreate` to handle the error (line 41)**

Replace line 41:
```go
h.createDocumentsForSources(ctx, customerID, id, storageFileIDs, sourceURLs)
```

With:
```go
if err := h.createDocumentsForSources(ctx, customerID, id, storageFileIDs, sourceURLs); err != nil {
	// Rollback: delete the just-created RAG since sources failed
	if delErr := h.dbHandler.RagDelete(ctx, id); delErr != nil {
		log.Errorf("Could not rollback rag creation. err: %v", delErr)
	}
	return nil, err
}
```

**Step 4: Update `RagAddSources` to handle the error (line 176)**

Replace line 176:
```go
h.createDocumentsForSources(ctx, r.CustomerID, r.ID, storageFileIDs, sourceURLs)
```

With:
```go
if err := h.createDocumentsForSources(ctx, r.CustomerID, r.ID, storageFileIDs, sourceURLs); err != nil {
	return nil, err
}
```

---

### Task 3: Add unit tests

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/rag_test.go` (append tests)

**Step 1: Add tests for duplicate detection**

Append the following tests to `rag_test.go`:

```go
func Test_RagAddSources_RejectsIntraRequestDuplicateFileIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := &ragHandler{dbHandler: mockDB}

	ragID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{
		ID:         ragID,
		CustomerID: uuid.Must(uuid.NewV4()),
	}, nil)

	_, err := h.RagAddSources(context.Background(), ragID, []uuid.UUID{fileID, fileID}, nil)
	if err == nil {
		t.Fatal("expected error for duplicate file IDs in request")
	}
}

func Test_RagAddSources_RejectsIntraRequestDuplicateURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := &ragHandler{dbHandler: mockDB}

	ragID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{
		ID:         ragID,
		CustomerID: uuid.Must(uuid.NewV4()),
	}, nil)

	_, err := h.RagAddSources(context.Background(), ragID, nil, []string{"https://example.com/doc.md", "https://example.com/doc.md"})
	if err == nil {
		t.Fatal("expected error for duplicate URLs in request")
	}
}

func Test_RagAddSources_RejectsExistingDuplicateSource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := &ragHandler{dbHandler: mockDB}

	ragID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{
		ID:         ragID,
		CustomerID: customerID,
	}, nil)

	// DB returns an existing document with the same file ID
	mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), ragID, []uuid.UUID{fileID}, []string(nil)).Return([]*document.Document{
		{ID: uuid.Must(uuid.NewV4()), StorageFileID: fileID, RagID: ragID},
	}, nil)

	_, err := h.RagAddSources(context.Background(), ragID, []uuid.UUID{fileID}, nil)
	if err == nil {
		t.Fatal("expected error for existing duplicate source")
	}
}

func Test_RagAddSources_AllowsNonDuplicateSources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := &ragHandler{
		dbHandler: mockDB,
		ingestSem: make(chan struct{}, maxConcurrentIngestions),
	}

	ragID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())
	docID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{
		ID:         ragID,
		CustomerID: customerID,
	}, nil)

	// No duplicates found
	mockDB.EXPECT().DocumentGetsByRagIDAndSources(gomock.Any(), ragID, []uuid.UUID{fileID}, []string(nil)).Return([]*document.Document{}, nil)

	// Document creation
	mockDB.EXPECT().DocumentCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().DocumentGet(gomock.Any(), gomock.Any()).Return(&document.Document{
		ID:            docID,
		CustomerID:    customerID,
		RagID:         ragID,
		StorageFileID: fileID,
		Status:        document.StatusPending,
	}, nil).Times(2) // Once for documentCreateInternal, once for RagGet enrichment

	// RagGet enrichment after source addition
	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{ID: ragID, CustomerID: customerID}, nil)
	mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), ragID).Return([]*document.Document{
		{ID: docID, StorageFileID: fileID, Status: document.StatusPending},
	}, nil)

	// Ingestion goroutine — may or may not fire during the test
	mockDB.EXPECT().DocumentClaimForProcessing(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("test skip")).AnyTimes()

	result, err := h.RagAddSources(context.Background(), ragID, []uuid.UUID{fileID}, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	time.Sleep(50 * time.Millisecond)
}
```

**Step 2: Run tests**

Run: `cd bin-rag-manager && go test ./...`
Expected: All tests pass.

---

### Task 4: Verify and commit

**Step 1: Run full verification workflow**

```bash
cd bin-rag-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

**Step 2: Commit**

```bash
git add -A ':!vendor'
git commit -m "NOJIRA-Reject-duplicate-rag-sources

Reject requests that add duplicate sources to a RAG. Detection happens
at two levels: intra-request (duplicate file IDs or URLs within the
same request) and DB-level (sources already active in the RAG).

- bin-rag-manager: Add DocumentGetsByRagIDAndSources DB method for duplicate lookup
- bin-rag-manager: Add duplicate detection in createDocumentsForSources before document creation
- bin-rag-manager: Change createDocumentsForSources to return error, propagate through RagCreate and RagAddSources
- bin-rag-manager: Add unit tests for intra-request and DB-level duplicate rejection"
```

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Reject-duplicate-rag-sources
```
