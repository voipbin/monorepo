package dbhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ParticipantListByAIcallID(t *testing.T) {
	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiID1 := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	aiID2 := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	otherAicallID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")

	curTime := time.Now()

	tests := []struct {
		name              string
		aicallID          uuid.UUID
		pageSize          uint64
		pageToken         string
		responseCurTime   *time.Time
		expectResultCount int
	}{
		{
			name:              "returns participants for aicall",
			aicallID:          aicallID,
			pageSize:          100,
			pageToken:         "",
			responseCurTime:   &curTime,
			expectResultCount: 2,
		},
		{
			name:              "empty token uses current time",
			aicallID:          aicallID,
			pageSize:          1,
			pageToken:         "",
			responseCurTime:   &curTime,
			expectResultCount: 1,
		},
		{
			name:              "other aicall returns empty",
			aicallID:          otherAicallID,
			pageSize:          100,
			pageToken:         "",
			responseCurTime:   &curTime,
			expectResultCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				_, _ = dbTest.Exec("DELETE FROM ai_aicall_participants WHERE aicall_id = ?", aicallID.Bytes())
			})

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{utilHandler: mockUtil, db: dbTest}

			// seed two participants for aicallID
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()
			_ = h.ParticipantCreate(context.Background(), aicallID, aiID1)
			_ = h.ParticipantCreate(context.Background(), aicallID, aiID2)

			mockUtil.EXPECT().TimeGetCurTime().Return("9999-12-31T23:59:59.999999Z").AnyTimes()

			res, err := h.ParticipantListByAIcallID(context.Background(), tt.aicallID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(res) != tt.expectResultCount {
				t.Fatalf("expected %d results, got %d", tt.expectResultCount, len(res))
			}
		})
	}
}

func Test_ParticipantListByAIID(t *testing.T) {
	aicallID1 := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	aicallID2 := uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666")
	aiID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")
	otherAIID := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")

	curTime := time.Now()

	tests := []struct {
		name              string
		aiID              uuid.UUID
		pageSize          uint64
		pageToken         string
		responseCurTime   *time.Time
		expectResultCount int
	}{
		{
			name:              "returns participants for ai",
			aiID:              aiID,
			pageSize:          100,
			pageToken:         "",
			responseCurTime:   &curTime,
			expectResultCount: 2,
		},
		{
			name:              "other ai returns empty",
			aiID:              otherAIID,
			pageSize:          100,
			pageToken:         "",
			responseCurTime:   &curTime,
			expectResultCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				_, _ = dbTest.Exec("DELETE FROM ai_aicall_participants WHERE ai_id = ?", aiID.Bytes())
			})

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := handler{utilHandler: mockUtil, db: dbTest}

			// seed two aicalls for aiID
			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime).AnyTimes()
			_ = h.ParticipantCreate(context.Background(), aicallID1, aiID)
			_ = h.ParticipantCreate(context.Background(), aicallID2, aiID)

			mockUtil.EXPECT().TimeGetCurTime().Return("9999-12-31T23:59:59.999999Z").AnyTimes()

			res, err := h.ParticipantListByAIID(context.Background(), tt.aiID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if len(res) != tt.expectResultCount {
				t.Fatalf("expected %d results, got %d", tt.expectResultCount, len(res))
			}
		})
	}
}

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
			t.Cleanup(func() {
				_, _ = dbTest.Exec("DELETE FROM ai_aicall_participants WHERE aicall_id = ?", tt.input.aicallID.Bytes())
			})

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
			}

			if tt.name == "duplicate insert is silently ignored" {
				// Pre-seed the row so this sub-test is self-contained and does not
				// depend on the "creates participant successfully" case running first.
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
				_ = h.ParticipantCreate(context.Background(), tt.input.aicallID, tt.input.aiID)
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
