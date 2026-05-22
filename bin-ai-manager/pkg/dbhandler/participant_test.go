package dbhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ParticipantCreate(t *testing.T) {
	curTime := time.Now()

	type input struct {
		aicallID uuid.UUID
		aiID     uuid.UUID
	}

	tests := []struct {
		name            string
		input           input
		responseCurTime *time.Time
		expectErr       bool
	}{
		{
			name: "creates participant successfully",
			input: input{
				aicallID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				aiID:     uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			},
			responseCurTime: &curTime,
			expectErr:       false,
		},
		{
			name: "duplicate insert is silently ignored",
			input: input{
				aicallID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				aiID:     uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			},
			responseCurTime: &curTime,
			expectErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)

			err := h.ParticipantCreate(context.Background(), tt.input.aicallID, tt.input.aiID)
			if tt.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}
