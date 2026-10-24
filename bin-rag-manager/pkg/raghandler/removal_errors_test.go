package raghandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/pkg/dbhandler"
)

// These cases cover the error exits that return before the background hard-delete goroutine
// is started, so no goroutine is involved and no mock call can outlive the test.

func Test_RagRemoveSource_error_exits(t *testing.T) {
	otherRagID := covDocID

	tests := []struct {
		name    string
		setup   func(db *dbhandler.MockDBHandler)
		wantErr string
		wantIs  error
	}{
		{
			name: "source lookup failure",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().DocumentGet(gomock.Any(), covFileID).Return(nil, dbhandler.ErrNotFound)
			},
			wantErr: "could not get source",
			wantIs:  dbhandler.ErrNotFound,
		},
		{
			name: "source belongs to another rag, nothing is deleted",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().DocumentGet(gomock.Any(), covFileID).Return(&document.Document{ID: covFileID, RagID: otherRagID}, nil)
			},
			wantErr: "source does not belong to this rag",
		},
		{
			name: "chunk soft delete failure keeps the document",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().DocumentGet(gomock.Any(), covFileID).Return(&document.Document{ID: covFileID, RagID: covRagID}, nil)
				db.EXPECT().ChunkSoftDeleteByDocumentID(gomock.Any(), covFileID).Return(errors.New("chunk delete failed"))
			},
			wantErr: "could not delete source chunks",
		},
		{
			name: "document delete failure",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().DocumentGet(gomock.Any(), covFileID).Return(&document.Document{ID: covFileID, RagID: covRagID}, nil)
				db.EXPECT().ChunkSoftDeleteByDocumentID(gomock.Any(), covFileID).Return(nil)
				db.EXPECT().DocumentDelete(gomock.Any(), covFileID).Return(errors.New("document delete failed"))
			},
			wantErr: "could not delete source:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB}
			tt.setup(mockDB)

			got, err := h.RagRemoveSource(context.Background(), covRagID, covFileID)
			if got != nil || err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Wrong match. expect nil and error containing %q, got: %v, %v", tt.wantErr, got, err)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("Wrong match. expect the cause %v to be wrapped, got: %v", tt.wantIs, err)
			}
		})
	}
}

func Test_RagDelete_error_exits(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(db *dbhandler.MockDBHandler)
		wantErr string
	}{
		{
			name: "chunk soft delete failure stops the cascade",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().ChunkSoftDeleteByRagID(gomock.Any(), covRagID).Return(errors.New("boom"))
			},
			wantErr: "could not delete rag chunks",
		},
		{
			name: "document delete failure leaves the rag in place",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().ChunkSoftDeleteByRagID(gomock.Any(), covRagID).Return(nil)
				db.EXPECT().DocumentDeleteByRagID(gomock.Any(), covRagID).Return(errors.New("boom"))
			},
			wantErr: "could not delete rag documents",
		},
		{
			name: "rag delete failure",
			setup: func(db *dbhandler.MockDBHandler) {
				db.EXPECT().ChunkSoftDeleteByRagID(gomock.Any(), covRagID).Return(nil)
				db.EXPECT().DocumentDeleteByRagID(gomock.Any(), covRagID).Return(nil)
				db.EXPECT().RagDelete(gomock.Any(), covRagID).Return(errors.New("boom"))
			},
			wantErr: "could not delete rag:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &ragHandler{dbHandler: mockDB}
			tt.setup(mockDB)

			err := h.RagDelete(context.Background(), covRagID)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Wrong match. expect error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
