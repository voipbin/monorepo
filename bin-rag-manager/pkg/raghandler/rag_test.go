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
