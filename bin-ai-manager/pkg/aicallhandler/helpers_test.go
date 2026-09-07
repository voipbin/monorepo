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
	"monorepo/bin-ai-manager/models/message"
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

// Test_insightSessionStart pins the boundary reader (VOIP-1484). Every failure
// mode must read as ABSENT, never as a zero time: a zero boundary would compare
// as older than every row and silently turn the cut into a no-op that looks
// like it worked, while a panic here would break every history rebuild.
func Test_insightSessionStart(t *testing.T) {
	boundary := time.Date(2026, 9, 7, 12, 0, 0, 123456789, time.UTC)

	tests := []struct {
		name string

		aicall *aicall.AIcall

		expectRes   time.Time
		expectFound bool
	}{
		{
			name: "nil aicall",

			aicall: nil,
		},
		{
			name: "nil metadata",

			aicall: &aicall.AIcall{},
		},
		{
			name: "absent key",

			aicall: &aicall.AIcall{Metadata: map[string]any{"other": 1}},
		},
		{
			name: "wrong type",

			aicall: &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyInsightSessionStart: 42}},
		},
		{
			name: "empty string",

			aicall: &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyInsightSessionStart: ""}},
		},
		{
			name: "unparsable string",

			aicall: &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyInsightSessionStart: "yesterday"}},
		},
		{
			name: "rfc3339 nano round-trips",

			aicall: &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyInsightSessionStart: boundary.Format(time.RFC3339Nano)}},

			expectRes:   boundary,
			expectFound: true,
		},
		{
			// Metadata round-trips through JSON and the writer stamps UTC, but a
			// value carrying an offset must still compare correctly.
			name: "a non-utc offset is normalised to utc",

			aicall: &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyInsightSessionStart: boundary.In(time.FixedZone("KST", 9*3600)).Format(time.RFC3339Nano)}},

			expectRes:   boundary,
			expectFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, found := insightSessionStart(tt.aicall)
			if found != tt.expectFound {
				t.Fatalf("found mismatch. expect: %v, got: %v", tt.expectFound, found)
			}
			if !found {
				return
			}
			if !res.Equal(tt.expectRes) {
				t.Errorf("boundary mismatch. expect: %s, got: %s", tt.expectRes, res)
			}
			if res.Location() != time.UTC {
				t.Errorf("the boundary must be returned in utc. got: %s", res.Location())
			}
		})
	}
}

// Test_cutBeforeSessionStart pins the replay cut shared by both history
// builders. The STRICT comparison is the load-bearing part: the boundary row IS
// the current session's first system row, so a `!After` predicate here would
// drop the Insight guardrails from every refreshed session.
func Test_cutBeforeSessionStart(t *testing.T) {
	boundary := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	before := boundary.Add(-time.Second)
	after := boundary.Add(time.Second)

	withBoundary := &aicall.AIcall{
		Metadata: map[string]any{aicall.MetaKeyInsightSessionStart: boundary.Format(time.RFC3339Nano)},
	}

	tests := []struct {
		name string

		rows   []*message.Message
		aicall *aicall.AIcall

		expectContents []string
	}{
		{
			name: "no boundary keeps every row",

			rows:   []*message.Message{{Content: "old", TMCreate: &before}, {Content: "new", TMCreate: &after}},
			aicall: &aicall.AIcall{},

			expectContents: []string{"old", "new"},
		},
		{
			name: "rows strictly before the boundary are dropped",

			rows:   []*message.Message{{Content: "old", TMCreate: &before}, {Content: "new", TMCreate: &after}},
			aicall: withBoundary,

			expectContents: []string{"new"},
		},
		{
			name: "the boundary row itself is kept",

			rows:   []*message.Message{{Content: "boundary", TMCreate: &boundary}},
			aicall: withBoundary,

			expectContents: []string{"boundary"},
		},
		{
			name: "a nil tm_create is kept",

			rows:   []*message.Message{{Content: "undated", TMCreate: nil}},
			aicall: withBoundary,

			expectContents: []string{"undated"},
		},
		{
			name: "everything older leaves an empty slice, never nil-panics",

			rows:   []*message.Message{{Content: "old", TMCreate: &before}},
			aicall: withBoundary,

			expectContents: []string{},
		},
		{
			name: "an empty input stays empty",

			rows:   []*message.Message{},
			aicall: withBoundary,

			expectContents: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := cutBeforeSessionStart(tt.rows, tt.aicall)

			contents := []string{}
			for _, m := range res {
				contents = append(contents, m.Content)
			}
			if len(contents) != len(tt.expectContents) {
				t.Fatalf("row count mismatch. expect: %v, got: %v", tt.expectContents, contents)
			}
			for i, want := range tt.expectContents {
				if contents[i] != want {
					t.Errorf("row %d mismatch. expect: %q, got: %q", i, want, contents[i])
				}
			}
		})
	}
}
