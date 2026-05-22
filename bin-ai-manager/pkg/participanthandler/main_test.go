package participanthandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/participant"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	dbErr := errors.New("db error")

	tests := []struct {
		name      string
		aicallID  uuid.UUID
		aiID      uuid.UUID
		mockSetup func(mockDB *dbhandler.MockDBHandler)
		expectErr bool
	}{
		{
			name:     "creates participant successfully",
			aicallID: aicallID,
			aiID:     aiID,
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantCreate(gomock.Any(), aicallID, aiID).Return(nil).Times(1)
			},
			expectErr: false,
		},
		{
			name:     "db error propagates",
			aicallID: aicallID,
			aiID:     aiID,
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantCreate(gomock.Any(), aicallID, aiID).Return(dbErr).Times(1)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			tt.mockSetup(mockDB)

			h := New(mockDB)
			err := h.Create(context.Background(), tt.aicallID, tt.aiID)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func Test_ListByAIcallID(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	dbErr := errors.New("db error")

	row := &participant.Participant{
		AIID:     aiID,
		AIcallID: aicallID,
	}

	tests := []struct {
		name      string
		aicallID  uuid.UUID
		pageSize  uint64
		pageToken string
		mockSetup func(mockDB *dbhandler.MockDBHandler)
		expectErr bool
		expectLen int
	}{
		{
			name:      "delegates to dbhandler",
			aicallID:  aicallID,
			pageSize:  100,
			pageToken: "",
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantListByAIcallID(gomock.Any(), aicallID, uint64(100), "").Return([]*participant.Participant{row}, nil).Times(1)
			},
			expectErr: false,
			expectLen: 1,
		},
		{
			name:      "db error propagates",
			aicallID:  aicallID,
			pageSize:  100,
			pageToken: "",
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantListByAIcallID(gomock.Any(), aicallID, uint64(100), "").Return(nil, dbErr).Times(1)
			},
			expectErr: true,
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			tt.mockSetup(mockDB)

			h := New(mockDB)
			res, err := h.ListByAIcallID(context.Background(), tt.aicallID, tt.pageSize, tt.pageToken)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(res) != tt.expectLen {
				t.Fatalf("expected %d results, got %d", tt.expectLen, len(res))
			}
		})
	}
}

func Test_ListByAIID(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	dbErr := errors.New("db error")

	row := &participant.Participant{
		AIID:     aiID,
		AIcallID: aicallID,
	}

	tests := []struct {
		name      string
		aiID      uuid.UUID
		pageSize  uint64
		pageToken string
		mockSetup func(mockDB *dbhandler.MockDBHandler)
		expectErr bool
		expectLen int
	}{
		{
			name:      "delegates to dbhandler",
			aiID:      aiID,
			pageSize:  100,
			pageToken: "",
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantListByAIID(gomock.Any(), aiID, uint64(100), "").Return([]*participant.Participant{row}, nil).Times(1)
			},
			expectErr: false,
			expectLen: 1,
		},
		{
			name:      "db error propagates",
			aiID:      aiID,
			pageSize:  100,
			pageToken: "",
			mockSetup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().ParticipantListByAIID(gomock.Any(), aiID, uint64(100), "").Return(nil, dbErr).Times(1)
			},
			expectErr: true,
			expectLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			tt.mockSetup(mockDB)

			h := New(mockDB)
			res, err := h.ListByAIID(context.Background(), tt.aiID, tt.pageSize, tt.pageToken)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(res) != tt.expectLen {
				t.Fatalf("expected %d results, got %d", tt.expectLen, len(res))
			}
		})
	}
}
