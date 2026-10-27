package dbhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-ai-manager/models/mcpoauthstate"
	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/models/team"
)

// Test_ClosedDatabase_ReturnsError verifies that every dbhandler operation
// reports an error, and does not touch the cache, when the database refuses the
// statement (here: the connection pool is closed).
func Test_ClosedDatabase_ReturnsError(t *testing.T) {
	h, _, mockCache := newCov90Handler(t, newClosedDB(t))

	errMiss := errors.New("cache miss")
	mockCache.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().AIcallGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().AIcallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().MessageGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().SummaryGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()
	mockCache.EXPECT().TeamGet(gomock.Any(), gomock.Any()).Return(nil, errMiss).AnyTimes()

	ctx := context.Background()
	id := uuid.FromStringOrNil("5c9d8f40-7a31-11f1-9b22-4f1c0a6e3d11")
	other := uuid.FromStringOrNil("5c9d8f41-7a31-11f1-9b22-4f1c0a6e3d11")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"AICreate", func() error { return h.AICreate(ctx, &ai.AI{}) }},
		{"AIGet", func() error { _, err := h.AIGet(ctx, id); return err }},
		{"AIList", func() error { _, err := h.AIList(ctx, 10, "", nil); return err }},
		{"AIUpdate", func() error { return h.AIUpdate(ctx, id, map[ai.Field]any{ai.FieldName: "n"}) }},
		{"AIDelete", func() error { return h.AIDelete(ctx, id) }},
		{"AIActivateInsight", func() error { _, _, err := h.AIActivateInsight(ctx, id); return err }},

		{"AIcallCreate", func() error { return h.AIcallCreate(ctx, &aicall.AIcall{}) }},
		{"AIcallGet", func() error { _, err := h.AIcallGet(ctx, id); return err }},
		{"AIcallGetSkipCache", func() error { _, err := h.AIcallGetSkipCache(ctx, id); return err }},
		{"AIcallGetByReferenceID", func() error { _, err := h.AIcallGetByReferenceID(ctx, id); return err }},
		{"AIcallGetByCustomerIDAndReferenceID", func() error { _, err := h.AIcallGetByCustomerIDAndReferenceID(ctx, id, other); return err }},
		{"AIcallList", func() error { _, err := h.AIcallList(ctx, 10, "", nil); return err }},
		{"AIcallUpdate", func() error {
			return h.AIcallUpdate(ctx, id, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing})
		}},
		{"AIcallUpdateNoTouchTMUpdate", func() error {
			return h.AIcallUpdateNoTouchTMUpdate(ctx, id, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing})
		}},
		{"AIcallUpdateIfActive", func() error {
			_, err := h.AIcallUpdateIfActive(ctx, id, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing})
			return err
		}},
		{"AIcallDelete", func() error { return h.AIcallDelete(ctx, id) }},

		{"AIAuditUpsert", func() error { _, err := h.AIAuditUpsert(ctx, &aiaudit.AIAudit{}); return err }},
		{"AIAuditGet", func() error { _, err := h.AIAuditGet(ctx, id); return err }},
		{"AIAuditList", func() error { _, err := h.AIAuditList(ctx, 10, "", nil); return err }},
		{"AIAuditDelete", func() error { return h.AIAuditDelete(ctx, id) }},
		{"AIAuditUpdateFinal", func() error {
			_, err := h.AIAuditUpdateFinal(ctx, id, aiaudit.StatusCompleted, nil, nil, "", nil)
			return err
		}},
		{"AIAuditCountProgressing", func() error { _, err := h.AIAuditCountProgressing(ctx, id); return err }},

		{"AIPromptHistoryCreate", func() error { return h.AIPromptHistoryCreate(ctx, &aiprompthistory.AIPromptHistory{}) }},
		{"AIPromptHistoryGet", func() error { _, err := h.AIPromptHistoryGet(ctx, id); return err }},
		{"AIPromptHistoryGetsByAIID", func() error { _, err := h.AIPromptHistoryGetsByAIID(ctx, id, 10, ""); return err }},

		{"AIPromptProposalCreate", func() error { return h.AIPromptProposalCreate(ctx, &aipromptproposal.AIPromptProposal{}) }},
		{"AIPromptProposalGet", func() error { _, err := h.AIPromptProposalGet(ctx, id); return err }},
		{"AIPromptProposalList", func() error { _, err := h.AIPromptProposalList(ctx, 10, "", nil); return err }},
		{"AIPromptProposalUpdateFinal", func() error {
			_, err := h.AIPromptProposalUpdateFinal(ctx, id, aipromptproposal.StatusCompleted, "p", "r", "")
			return err
		}},
		{"AIPromptProposalUpdateExpired", func() error { _, err := h.AIPromptProposalUpdateExpired(ctx, id, "e"); return err }},
		{"AIPromptProposalUpdateRejected", func() error { _, err := h.AIPromptProposalUpdateRejected(ctx, id); return err }},
		{"AIPromptProposalDelete", func() error { return h.AIPromptProposalDelete(ctx, id) }},
		{"AIPromptProposalCountProgressing", func() error { _, err := h.AIPromptProposalCountProgressing(ctx, id); return err }},
		{"AIAcceptProposal", func() error { return h.AIAcceptProposal(ctx, id, other, "prompt") }},

		{"McpOAuthStateCreate", func() error { return h.McpOAuthStateCreate(ctx, &mcpoauthstate.McpOAuthState{}) }},
		{"McpOAuthStateGet", func() error { _, err := h.McpOAuthStateGet(ctx, "state"); return err }},
		{"McpOAuthStateDelete", func() error { return h.McpOAuthStateDelete(ctx, "state") }},
		{"McpOAuthStateDeleteExpired", func() error { _, err := h.McpOAuthStateDeleteExpired(ctx, "2026-01-01 00:00:00.000000"); return err }},

		{"McpServerCreate", func() error { return h.McpServerCreate(ctx, &mcpserver.McpServer{}) }},
		{"McpServerGet", func() error { _, err := h.McpServerGet(ctx, id); return err }},
		{"McpServerList", func() error { _, err := h.McpServerList(ctx, 10, "", nil); return err }},
		{"McpServerUpdate", func() error { return h.McpServerUpdate(ctx, id, map[mcpserver.Field]any{mcpserver.FieldName: "n"}) }},
		{"McpServerUpdateOAuthTokensIfCurrent", func() error {
			return h.McpServerUpdateOAuthTokensIfCurrent(ctx, id, []byte("old"), map[mcpserver.Field]any{mcpserver.FieldName: "n"})
		}},
		{"McpServerDelete", func() error { return h.McpServerDelete(ctx, id) }},

		{"MessageCreate", func() error { return h.MessageCreate(ctx, &message.Message{}) }},
		{"MessageGet", func() error { _, err := h.MessageGet(ctx, id); return err }},
		{"MessageList", func() error { _, err := h.MessageList(ctx, 10, "", nil); return err }},
		{"MessageDelete", func() error { return h.MessageDelete(ctx, id) }},
		{"MessageUpdateDeliveryStatus", func() error { return h.MessageUpdateDeliveryStatus(ctx, id, message.DeliveryStatus("sent")) }},
		{"MessageAssistantReplyExists", func() error { _, err := h.MessageAssistantReplyExists(ctx, id); return err }},

		{"ParticipantCreate", func() error { return h.ParticipantCreate(ctx, id, other) }},
		{"ParticipantListByAIcallID", func() error { _, err := h.ParticipantListByAIcallID(ctx, id, 10, ""); return err }},
		{"ParticipantListByAIID", func() error { _, err := h.ParticipantListByAIID(ctx, id, 10, ""); return err }},

		{"SummaryCreate", func() error { return h.SummaryCreate(ctx, &summary.Summary{}) }},
		{"SummaryGet", func() error { _, err := h.SummaryGet(ctx, id); return err }},
		{"SummaryList", func() error { _, err := h.SummaryList(ctx, 10, "", nil); return err }},
		{"SummaryUpdate", func() error { return h.SummaryUpdate(ctx, id, map[summary.Field]any{summary.FieldContent: "c"}) }},
		{"SummaryUpdateStatusDoneIfNotDone", func() error { _, err := h.SummaryUpdateStatusDoneIfNotDone(ctx, id, "c"); return err }},
		{"SummaryDelete", func() error { return h.SummaryDelete(ctx, id) }},

		{"TeamCreate", func() error { return h.TeamCreate(ctx, &team.Team{}) }},
		{"TeamGet", func() error { _, err := h.TeamGet(ctx, id); return err }},
		{"TeamList", func() error { _, err := h.TeamList(ctx, 10, "", nil); return err }},
		{"TeamUpdate", func() error { return h.TeamUpdate(ctx, id, map[team.Field]any{team.FieldName: "n"}) }},
		{"TeamDelete", func() error { return h.TeamDelete(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Errorf("%s: expected an error from a closed database, got nil", tt.name)
			}
		})
	}
}
