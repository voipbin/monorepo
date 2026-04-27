package aicallhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_aicallHandler_isAIcallReusable(t *testing.T) {
	config.SetAIcallConversationIdleTimeoutHoursForTest(24)

	fresh := time.Now().Add(-1 * time.Hour)
	expired := time.Now().Add(-25 * time.Hour)

	tests := []struct {
		name string

		ac        *aicall.AIcall
		expectRes bool
	}{
		{
			name: "nil",

			ac:        nil,
			expectRes: false,
		},
		{
			name: "progressing fresh",

			ac:        &aicall.AIcall{Status: aicall.StatusProgressing, TMUpdate: &fresh},
			expectRes: true,
		},
		{
			name: "initiating fresh",

			ac:        &aicall.AIcall{Status: aicall.StatusInitiating, TMUpdate: &fresh},
			expectRes: true,
		},
		{
			name: "terminated",

			ac:        &aicall.AIcall{Status: aicall.StatusTerminated, TMUpdate: &fresh},
			expectRes: false,
		},
		{
			name: "terminating",

			ac:        &aicall.AIcall{Status: aicall.StatusTerminating, TMUpdate: &fresh},
			expectRes: false,
		},
		{
			name: "idle expired",

			ac:        &aicall.AIcall{Status: aicall.StatusProgressing, TMUpdate: &expired},
			expectRes: false,
		},
		{
			name: "nil tm_update",

			ac:        &aicall.AIcall{Status: aicall.StatusProgressing},
			expectRes: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := &aicallHandler{}

			got := h.isAIcallReusable(tt.ac)
			if got != tt.expectRes {
				t.Errorf("expected: %v, got: %v", tt.expectRes, got)
			}
		})
	}
}

func Test_aicallHandler_isAIcallIdleExpired(t *testing.T) {
	config.SetAIcallConversationIdleTimeoutHoursForTest(24)

	twentyThreeHoursAgo := time.Now().Add(-23 * time.Hour)
	twentyFiveHoursAgo := time.Now().Add(-25 * time.Hour)

	tests := []struct {
		name string

		ac        *aicall.AIcall
		expectRes bool
	}{
		{
			name: "nil",

			ac:        nil,
			expectRes: false,
		},
		{
			name: "nil tm_update",

			ac:        &aicall.AIcall{Status: aicall.StatusProgressing},
			expectRes: false,
		},
		{
			name: "23h ago is under 24h",

			ac:        &aicall.AIcall{Status: aicall.StatusProgressing, TMUpdate: &twentyThreeHoursAgo},
			expectRes: false,
		},
		{
			name: "25h ago is over 24h",

			ac:        &aicall.AIcall{Status: aicall.StatusProgressing, TMUpdate: &twentyFiveHoursAgo},
			expectRes: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := &aicallHandler{}

			got := h.isAIcallIdleExpired(tt.ac)
			if got != tt.expectRes {
				t.Errorf("expected: %v, got: %v", tt.expectRes, got)
			}
		})
	}
}

func Test_aicallHandler_interruptPreviousPipecatcall(t *testing.T) {
	pcID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string

		pcID      uuid.UUID
		mockSetup func(rh *requesthandler.MockRequestHandler)
	}{
		{
			name: "nil pcID — no calls made",

			pcID: uuid.Nil,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				// no expectations — must NOT be called
			},
		},
		{
			name: "Get fails — no Ping or Terminate",

			pcID: pcID,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().
					PipecatV1PipecatcallGet(gomock.Any(), pcID).
					Return(nil, errors.New("not found"))
			},
		},
		{
			name: "ping returns dead — no Terminate",

			pcID: pcID,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				pc := &pmpipecatcall.Pipecatcall{
					Identity: identity.Identity{ID: pcID},
					HostID:   "10.0.0.1",
				}
				rh.EXPECT().
					PipecatV1PipecatcallGet(gomock.Any(), pcID).
					Return(pc, nil)
				rh.EXPECT().
					PipecatV1Ping(gomock.Any(), "10.0.0.1").
					Return(context.DeadlineExceeded)
			},
		},
		{
			name: "ping ok, terminate succeeds",

			pcID: pcID,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				pc := &pmpipecatcall.Pipecatcall{
					Identity: identity.Identity{ID: pcID},
					HostID:   "10.0.0.2",
				}
				rh.EXPECT().
					PipecatV1PipecatcallGet(gomock.Any(), pcID).
					Return(pc, nil)
				rh.EXPECT().
					PipecatV1Ping(gomock.Any(), "10.0.0.2").
					Return(nil)
				rh.EXPECT().
					PipecatV1PipecatcallTerminate(gomock.Any(), "10.0.0.2", pcID).
					Return(nil, nil)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			tt.mockSetup(mockReq)

			h := &aicallHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			h.interruptPreviousPipecatcall(ctx, tt.pcID)
			// no return value — gomock EXPECTs enforce correctness
		})
	}
}
