package participanthandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

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
