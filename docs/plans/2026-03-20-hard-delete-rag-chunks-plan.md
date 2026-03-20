# Hard-Delete RAG Chunks Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** After soft-deleting chunks, fire a background goroutine to hard-delete them, reclaiming storage from 768-dimension vector embeddings.

**Architecture:** Two-phase delete — soft-delete makes chunks invisible to search immediately, then a background goroutine calls the existing hard-delete DB methods. No new DB methods, no schema changes.

**Tech Stack:** Go, gomock, logrus, context.Background()

---

### Task 1: Add background hard-delete wrapper methods

**Files:**
- Modify: `bin-rag-manager/pkg/raghandler/rag.go:134-211`

**Step 1: Write the two private wrapper methods**

Add these two methods at the end of `rag.go` (before the `createDocumentsForSources` function at line 213):

```go
// chunkHardDeleteByDocumentID hard-deletes all chunks for a document in the background.
// Uses context.Background() because the request context may be cancelled after the response is sent.
// If the hard-delete fails, chunks remain soft-deleted (invisible to search) — safe degradation.
func (h *ragHandler) chunkHardDeleteByDocumentID(documentID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "chunkHardDeleteByDocumentID",
		"document_id": documentID,
	})

	if err := h.dbHandler.ChunkDeleteByDocumentID(context.Background(), documentID); err != nil {
		log.Errorf("Could not hard delete chunks for document. err: %v", err)
		return
	}
	log.Debugf("Hard deleted chunks for document. document_id: %s", documentID)
}

// chunkHardDeleteByRagID hard-deletes all chunks for a rag in the background.
// Uses context.Background() because the request context may be cancelled after the response is sent.
// If the hard-delete fails, chunks remain soft-deleted (invisible to search) — safe degradation.
func (h *ragHandler) chunkHardDeleteByRagID(ragID uuid.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "chunkHardDeleteByRagID",
		"rag_id": ragID,
	})

	if err := h.dbHandler.ChunkDeleteByRagID(context.Background(), ragID); err != nil {
		log.Errorf("Could not hard delete chunks for rag. err: %v", err)
		return
	}
	log.Debugf("Hard deleted chunks for rag. rag_id: %s", ragID)
}
```

**Step 2: Add goroutine call in `RagDelete`**

In `RagDelete` (line 134), add the goroutine call after the existing soft-delete cascade succeeds and before the return. Insert after line 155 (`log.Debugf("Deleted rag...")`):

```go
	// Background hard-delete of chunks to reclaim storage
	go h.chunkHardDeleteByRagID(id)
```

**Step 3: Add goroutine call in `RagRemoveSource`**

In `RagRemoveSource` (line 178), add the goroutine call after the document soft-delete succeeds. Insert after line 208 (`log.Debugf("Deleted source...")`):

```go
	// Background hard-delete of chunks to reclaim storage
	go h.chunkHardDeleteByDocumentID(sourceID)
```

**Step 4: Run tests to verify nothing is broken**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Hard-delete-rag-chunks-after-soft-delete/bin-rag-manager && go test ./...`
Expected: All existing tests PASS (no behavioral change to existing tests)

### Task 2: Add unit tests for background hard-delete

**Files:**
- Create: `bin-rag-manager/pkg/raghandler/rag_test.go`

**Step 1: Write tests for `RagDelete` and `RagRemoveSource` that verify hard-delete is called**

Since the hard-delete runs in a goroutine, the tests need a small sleep or sync mechanism to wait for the goroutine to complete. Use `time.Sleep(50 * time.Millisecond)` which is simple and sufficient for a unit test with mocked DB calls.

```go
package raghandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-rag-manager/pkg/dbhandler"
)

func Test_RagDelete_HardDeletesChunks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)

	h := &ragHandler{
		dbHandler: mockDB,
	}

	ragID := uuid.Must(uuid.NewV4())

	// Expect soft-delete cascade
	mockDB.EXPECT().ChunkSoftDeleteByRagID(gomock.Any(), ragID).Return(nil)
	mockDB.EXPECT().DocumentDeleteByRagID(gomock.Any(), ragID).Return(nil)
	mockDB.EXPECT().RagDelete(gomock.Any(), ragID).Return(nil)

	// Expect background hard-delete
	mockDB.EXPECT().ChunkDeleteByRagID(gomock.Any(), ragID).Return(nil)

	err := h.RagDelete(context.Background(), ragID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Wait for background goroutine to complete
	time.Sleep(50 * time.Millisecond)
}

func Test_RagDelete_HardDeleteFailure_DoesNotAffectSoftDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)

	h := &ragHandler{
		dbHandler: mockDB,
	}

	ragID := uuid.Must(uuid.NewV4())

	// Soft-delete cascade succeeds
	mockDB.EXPECT().ChunkSoftDeleteByRagID(gomock.Any(), ragID).Return(nil)
	mockDB.EXPECT().DocumentDeleteByRagID(gomock.Any(), ragID).Return(nil)
	mockDB.EXPECT().RagDelete(gomock.Any(), ragID).Return(nil)

	// Background hard-delete fails — should not affect the soft-delete result
	mockDB.EXPECT().ChunkDeleteByRagID(gomock.Any(), ragID).Return(fmt.Errorf("db connection lost"))

	err := h.RagDelete(context.Background(), ragID)
	if err != nil {
		t.Fatalf("expected no error from RagDelete even when hard-delete fails, got: %v", err)
	}

	// Wait for background goroutine to complete
	time.Sleep(50 * time.Millisecond)
}

func Test_RagRemoveSource_HardDeletesChunks(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)

	h := &ragHandler{
		dbHandler: mockDB,
	}

	ragID := uuid.Must(uuid.NewV4())
	sourceID := uuid.Must(uuid.NewV4())

	doc := &document.Document{
		ID:    sourceID,
		RagID: ragID,
	}

	// Expect document verification
	mockDB.EXPECT().DocumentGet(gomock.Any(), sourceID).Return(doc, nil)

	// Expect soft-delete cascade
	mockDB.EXPECT().ChunkSoftDeleteByDocumentID(gomock.Any(), sourceID).Return(nil)
	mockDB.EXPECT().DocumentDelete(gomock.Any(), sourceID).Return(nil)

	// Expect RagGet for return value enrichment
	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{ID: ragID}, nil)
	mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), ragID).Return([]*document.Document{}, nil)

	// Expect background hard-delete
	mockDB.EXPECT().ChunkDeleteByDocumentID(gomock.Any(), sourceID).Return(nil)

	_, err := h.RagRemoveSource(context.Background(), ragID, sourceID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Wait for background goroutine to complete
	time.Sleep(50 * time.Millisecond)
}

func Test_RagRemoveSource_HardDeleteFailure_DoesNotAffectSoftDelete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)

	h := &ragHandler{
		dbHandler: mockDB,
	}

	ragID := uuid.Must(uuid.NewV4())
	sourceID := uuid.Must(uuid.NewV4())

	doc := &document.Document{
		ID:    sourceID,
		RagID: ragID,
	}

	// Expect document verification
	mockDB.EXPECT().DocumentGet(gomock.Any(), sourceID).Return(doc, nil)

	// Soft-delete cascade succeeds
	mockDB.EXPECT().ChunkSoftDeleteByDocumentID(gomock.Any(), sourceID).Return(nil)
	mockDB.EXPECT().DocumentDelete(gomock.Any(), sourceID).Return(nil)

	// Expect RagGet for return value enrichment
	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{ID: ragID}, nil)
	mockDB.EXPECT().DocumentGetsByRagID(gomock.Any(), ragID).Return([]*document.Document{}, nil)

	// Background hard-delete fails — should not affect the result
	mockDB.EXPECT().ChunkDeleteByDocumentID(gomock.Any(), sourceID).Return(fmt.Errorf("db connection lost"))

	result, err := h.RagRemoveSource(context.Background(), ragID, sourceID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Wait for background goroutine to complete
	time.Sleep(50 * time.Millisecond)
}
```

**Step 2: Run the new tests**

Run: `cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Hard-delete-rag-chunks-after-soft-delete/bin-rag-manager && go test -v ./pkg/raghandler/...`
Expected: All 4 new tests PASS plus existing tests

### Task 3: Run full verification workflow and commit

**Step 1: Run the full verification workflow**

Run:
```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Hard-delete-rag-chunks-after-soft-delete/bin-rag-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All steps PASS with no errors

**Step 2: Commit**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Hard-delete-rag-chunks-after-soft-delete
git add docs/plans/2026-03-20-hard-delete-rag-chunks-design.md
git add docs/plans/2026-03-20-hard-delete-rag-chunks-plan.md
git add bin-rag-manager/pkg/raghandler/rag.go
git add bin-rag-manager/pkg/raghandler/rag_test.go
git commit -m "NOJIRA-Hard-delete-rag-chunks-after-soft-delete

Add background hard-delete of RAG chunks after soft-delete to reclaim
storage from 768-dimension vector embeddings.

- bin-rag-manager: Add chunkHardDeleteByDocumentID and chunkHardDeleteByRagID wrapper methods
- bin-rag-manager: Fire background goroutines in RagDelete and RagRemoveSource after soft-delete
- bin-rag-manager: Add unit tests for hard-delete in both RagDelete and RagRemoveSource
- docs: Add design document and implementation plan"
```

**Step 3: Push and create PR**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Hard-delete-rag-chunks-after-soft-delete
git push -u origin NOJIRA-Hard-delete-rag-chunks-after-soft-delete
```
