package messagehandler

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/message"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
)

// The SMS draft decision claim row (role notification, type draft_decision,
// VOIP-1704) is a notification of another type: the pipeline error scans must
// ignore it, and the unanswered-message query must not count it as the newest
// message of the session.
func Test_draftDecisionClaimRow_ignoredByPipelineErrorScans(t *testing.T) {
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	created := now.Add(-time.Minute)
	claim := &message.Message{
		Role:     message.RoleNotification,
		Content:  `{"type":"draft_decision","draft_id":"d9000000-0000-11f0-0000-000000000001","decision":"approve"}`,
		TMCreate: &created,
	}

	tests := []struct {
		name string
		run  func(m *pipelineErrorTestMocks) bool
	}{
		{
			name: "pipelineErrorNoticeBlocks",
			run: func(m *pipelineErrorTestMocks) bool {
				return m.h.pipelineErrorNoticeBlocks(t.Context(), testPEAicallID, testPEPipecatcallID, pmmessage.ErrorCategoryTimeout)
			},
		},
		{
			name: "pipelineErrorNoticeRecent",
			run: func(m *pipelineErrorTestMocks) bool {
				return m.h.pipelineErrorNoticeRecent(t.Context(), testPEAicallID, pmmessage.ErrorCategoryTimeout)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newPipelineErrorTestMocks(mc)

			m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), "", gomock.Any()).Return([]*message.Message{claim}, nil)
			m.util.EXPECT().TimeNow().Return(&now)

			if tt.run(m) {
				t.Errorf("a draft_decision claim row must not block a pipeline error notice")
			}
		})
	}
}

func Test_draftDecisionClaimRow_notTheNewestMessageForUnanswered(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newPipelineErrorTestMocks(mc)

	// the query excludes notification rows, so the claim cannot be returned;
	// the filter must be the same NotEq(notification) the timeout path uses.
	m.db.EXPECT().MessageList(gomock.Any(), uint64(1), "", latestNonNotificationFilter).Return([]*message.Message{{Role: message.RoleAssistant}}, nil)

	unanswered, err := m.h.userMessageUnanswered(t.Context(), testPEAicallID, uuid.Nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if unanswered {
		t.Errorf("expected not unanswered")
	}
}
