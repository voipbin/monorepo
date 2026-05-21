package aicallhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/teamhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_aicallHandler_resolveActiveAIIDFromAIcall(t *testing.T) {
	aiID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	teamID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")
	memberID := uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000003")
	aiidForMember := uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000004")

	tests := []struct {
		name      string
		ac        *aicall.AIcall
		mockSetup func(th *teamhandler.MockTeamHandler)
		want      uuid.UUID
	}{
		{
			name: "AI type — returns AssistanceID directly without RPC",
			ac: &aicall.AIcall{
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   aiID,
			},
			mockSetup: func(th *teamhandler.MockTeamHandler) {},
			want:      aiID,
		},
		{
			name: "Team type — member found",
			ac: &aicall.AIcall{
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    teamID,
				CurrentMemberID: memberID,
			},
			mockSetup: func(th *teamhandler.MockTeamHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(&team.Team{
					Members: []team.Member{
						{ID: memberID, AIID: aiidForMember},
					},
				}, nil)
			},
			want: aiidForMember,
		},
		{
			name: "Team type — member not found returns uuid.Nil",
			ac: &aicall.AIcall{
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    teamID,
				CurrentMemberID: uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-000000000005"),
			},
			mockSetup: func(th *teamhandler.MockTeamHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(&team.Team{
					Members: []team.Member{
						{ID: memberID, AIID: aiidForMember},
					},
				}, nil)
			},
			want: uuid.Nil,
		},
		{
			name: "Team type — TeamGet error returns uuid.Nil",
			ac: &aicall.AIcall{
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    teamID,
				CurrentMemberID: memberID,
			},
			mockSetup: func(th *teamhandler.MockTeamHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(nil, errors.New("not found"))
			},
			want: uuid.Nil,
		},
		{
			name: "Unknown AssistanceType returns uuid.Nil",
			ac: &aicall.AIcall{
				AssistanceType: "unknown",
				AssistanceID:   aiID,
			},
			mockSetup: func(th *teamhandler.MockTeamHandler) {},
			want:      uuid.Nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockTeam := teamhandler.NewMockTeamHandler(mc)
			tt.mockSetup(mockTeam)

			h := &aicallHandler{teamHandler: mockTeam}
			got := h.resolveActiveAIIDFromAIcall(context.Background(), tt.ac)
			if got != tt.want {
				t.Errorf("resolveActiveAIIDFromAIcall() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

		// expectLabel — the metric label expected to increment by 1.
		// Empty string means no metric should increment (e.g. nil pcID short-circuit).
		expectLabel string
	}{
		{
			name: "nil pcID — no calls made",

			pcID: uuid.Nil,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				// no expectations — must NOT be called
			},
			expectLabel: "",
		},
		{
			name: "Get fails — no Ping or Terminate",

			pcID: pcID,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				rh.EXPECT().
					PipecatV1PipecatcallGet(gomock.Any(), pcID).
					Return(nil, errors.New("not found"))
			},
			expectLabel: "gone",
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
			expectLabel: "dead",
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
			expectLabel: "alive",
		},
		{
			name: "ping ok, terminate fails",

			pcID: pcID,
			mockSetup: func(rh *requesthandler.MockRequestHandler) {
				pc := &pmpipecatcall.Pipecatcall{
					Identity: identity.Identity{ID: pcID},
					HostID:   "10.0.0.3",
				}
				rh.EXPECT().
					PipecatV1PipecatcallGet(gomock.Any(), pcID).
					Return(pc, nil)
				rh.EXPECT().
					PipecatV1Ping(gomock.Any(), "10.0.0.3").
					Return(nil)
				rh.EXPECT().
					PipecatV1PipecatcallTerminate(gomock.Any(), "10.0.0.3", pcID).
					Return(nil, errors.New("terminate failed"))
			},
			expectLabel: "error",
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

			// Snapshot all four labels before the call so we can verify
			// exactly one was incremented (or none, for the nil-pcID case).
			labels := []string{"gone", "dead", "alive", "error"}
			before := make(map[string]float64, len(labels))
			for _, l := range labels {
				before[l] = testutil.ToFloat64(promAIcallInterruptAttemptedTotal.WithLabelValues(l))
			}

			h.interruptPreviousPipecatcall(ctx, tt.pcID)
			// no return value — gomock EXPECTs enforce correctness

			for _, l := range labels {
				after := testutil.ToFloat64(promAIcallInterruptAttemptedTotal.WithLabelValues(l))
				delta := after - before[l]
				if l == tt.expectLabel {
					if delta != 1 {
						t.Errorf("expected label %q to increment by 1, got delta=%f", l, delta)
					}
				} else {
					if delta != 0 {
						t.Errorf("expected label %q to NOT change, got delta=%f", l, delta)
					}
				}
			}
		})
	}
}
