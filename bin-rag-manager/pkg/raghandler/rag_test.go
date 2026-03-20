package raghandler

import (
	"context"
	"fmt"
	"strings"
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

func Test_RagAddSources_RejectsIntraRequestDuplicateFileIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := &ragHandler{dbHandler: mockDB}

	ragID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().RagGet(gomock.Any(), ragID).Return(&rag.Rag{
		ID:         ragID,
		CustomerID: uuid.Must(uuid.NewV4()),
	}, nil)

	fileID := uuid.Must(uuid.NewV4())
	_, err := h.RagAddSources(context.Background(), ragID, []uuid.UUID{fileID, fileID}, nil)
	if err == nil {
		t.Fatal("expected error for duplicate file IDs in request")
	}
	if !strings.Contains(err.Error(), "duplicate source(s) in request") {
		t.Fatalf("unexpected error message: %v", err)
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
	if !strings.Contains(err.Error(), "duplicate source(s) in request") {
		t.Fatalf("unexpected error message: %v", err)
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
	if !strings.Contains(err.Error(), "source(s) already exist in this rag") {
		t.Fatalf("unexpected error message: %v", err)
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
	}, nil)

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
