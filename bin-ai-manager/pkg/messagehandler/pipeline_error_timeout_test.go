package messagehandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

func newContactCaseAIcall(status aicall.Status, pipecatcallID uuid.UUID) *aicall.AIcall {
	ac := newPipelineErrorAIcall(pipecatcallID)
	ac.CustomerID = testPECustomerID
	ac.ActiveflowID = testPEActiveflowID
	ac.ReferenceType = aicall.ReferenceTypeContactCase
	ac.Status = status
	return ac
}

func newTerminatedEvent() *pmpipecatcall.Pipecatcall {
	evt := &pmpipecatcall.Pipecatcall{
		ReferenceType: pmpipecatcall.ReferenceTypeAICall,
		ReferenceID:   testPEAicallID,
	}
	evt.ID = testPEPipecatcallID
	return evt
}

var latestNonNotificationFilter = map[message.Field]any{
	message.FieldAIcallID: testPEAicallID,
	message.FieldDeleted:  false,
	message.FieldRole:     commondatabasehandler.NotEq{Value: string(message.RoleNotification)},
}

var noticeScanFilter = map[message.Field]any{
	message.FieldAIcallID: testPEAicallID,
	message.FieldRole:     message.RoleNotification,
	message.FieldDeleted:  false,
}

func noticeRowFor(t *testing.T, category pmmessage.ErrorCategory, pipecatcallID uuid.UUID, created time.Time) *message.Message {
	content, err := json.Marshal(PipelineErrorNotice{Type: NotificationTypePipelineError, Category: category, PipecatcallID: pipecatcallID})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &message.Message{Role: message.RoleNotification, Content: string(content), TMCreate: &created}
}

func otherTypeNoticeRow(t *testing.T, category pmmessage.ErrorCategory, pipecatcallID uuid.UUID, created time.Time) *message.Message {
	content, err := json.Marshal(PipelineErrorNotice{Type: "member_switched", Category: category, PipecatcallID: pipecatcallID})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return &message.Message{Role: message.RoleNotification, Content: string(content), TMCreate: &created}
}

func Test_EventPMPipecatcallTerminated_contactCaseTimeoutNotice(t *testing.T) {
	now := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	userRow := []*message.Message{{Role: message.RoleUser}}

	tests := []struct {
		name string

		cached *aicall.AIcall // first (cached) read

		expectSleep bool

		fresh    *aicall.AIcall // SkipCache read result
		freshErr error

		expectLatest bool
		latest       []*message.Message
		latestErr    error

		expectReplyExists bool
		replyExists       bool
		replyExistsErr    error

		expectScan bool
		scan       []*message.Message
		scanErr    error

		expectCreate bool
		createErr    error
		wantLabel    string
	}{
		{
			name:              "unanswered user message creates a timeout notice",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			expectCreate:      true,
			wantLabel:         "timeout_notice_created",
		},
		{
			name:         "cached terminating is skipped before the sleep",
			cached:       newContactCaseAIcall(aicall.StatusTerminating, testPEPipecatcallID),
			expectSleep:  false,
			wantLabel:    "timeout_notice_skipped",
			expectCreate: false,
		},
		{
			name:        "cached terminated is skipped before the sleep",
			cached:      newContactCaseAIcall(aicall.StatusTerminated, testPEPipecatcallID),
			expectSleep: false,
			wantLabel:   "timeout_notice_skipped",
		},
		{
			name:        "listen turn or older pipecatcall (cached id differs) is skipped before the sleep",
			cached:      newContactCaseAIcall(aicall.StatusProgressing, testPEOtherPipecallID),
			expectSleep: false,
			wantLabel:   "timeout_notice_skipped",
		},
		{
			name:        "interrupt race: cache shows the old id, the fresh read shows the new turn",
			cached:      newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep: true,
			fresh:       newContactCaseAIcall(aicall.StatusProgressing, testPEOtherPipecallID),
			wantLabel:   "timeout_notice_skipped",
		},
		{
			name:        "fresh read shows terminating",
			cached:      newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep: true,
			fresh:       newContactCaseAIcall(aicall.StatusTerminating, testPEPipecatcallID),
			wantLabel:   "timeout_notice_skipped",
		},
		{
			name:        "fresh read error ends without a notice",
			cached:      newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep: true,
			freshErr:    errors.New("rpc down"),
		},
		{
			name:         "latest message is an assistant reply",
			cached:       newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:  true,
			fresh:        newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest: true,
			latest:       []*message.Message{{Role: message.RoleAssistant}},
			wantLabel:    "timeout_notice_skipped",
		},
		{
			name:         "latest message is a tool row (tools ran)",
			cached:       newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:  true,
			fresh:        newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest: true,
			latest:       []*message.Message{{Role: message.RoleTool}},
			wantLabel:    "timeout_notice_skipped",
		},
		{
			name:         "no messages at all",
			cached:       newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:  true,
			fresh:        newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest: true,
			latest:       []*message.Message{},
			wantLabel:    "timeout_notice_skipped",
		},
		{
			name:         "message list error ends without a notice",
			cached:       newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:  true,
			fresh:        newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest: true,
			latestErr:    errors.New("db down"),
		},
		{
			name:              "assistant reply exists for the pipecatcall",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			replyExists:       true,
			wantLabel:         "timeout_notice_skipped",
		},
		{
			name:              "reply exists error ends without a notice",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			replyExistsErr:    errors.New("db down"),
		},
		{
			name:              "a pipeline error notice already exists for this pipecatcall (any category)",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			scan:              []*message.Message{noticeRowFor(t, pmmessage.ErrorCategoryServerError, testPEPipecatcallID, now.Add(-time.Hour))},
			wantLabel:         "timeout_notice_skipped",
		},
		{
			name:              "a timeout notice of another turn within the window",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			scan:              []*message.Message{noticeRowFor(t, pmmessage.ErrorCategoryTimeout, testPEOtherPipecallID, now.Add(-5*time.Minute))},
			wantLabel:         "timeout_notice_skipped",
		},
		{
			name:              "a timeout notice of another turn outside the window",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			scan:              []*message.Message{noticeRowFor(t, pmmessage.ErrorCategoryTimeout, testPEOtherPipecallID, now.Add(-11*time.Minute))},
			expectCreate:      true,
			wantLabel:         "timeout_notice_created",
		},
		{
			name:              "another category of another turn within the window does not block",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			scan:              []*message.Message{noticeRowFor(t, pmmessage.ErrorCategoryAuthentication, testPEOtherPipecallID, now.Add(-1*time.Minute))},
			expectCreate:      true,
			wantLabel:         "timeout_notice_created",
		},
		{
			name:              "a notification of another type with the same turn and category does not block",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			scan:              []*message.Message{otherTypeNoticeRow(t, pmmessage.ErrorCategoryTimeout, testPEPipecatcallID, now.Add(-1*time.Minute))},
			expectCreate:      true,
			wantLabel:         "timeout_notice_created",
		},
		{
			name:              "scan error is fail-open",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			scanErr:           errors.New("db down"),
			expectCreate:      true,
			wantLabel:         "timeout_notice_created",
		},
		{
			name:              "create failure is counted and returns nil",
			cached:            newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectSleep:       true,
			fresh:             newContactCaseAIcall(aicall.StatusProgressing, testPEPipecatcallID),
			expectLatest:      true,
			latest:            userRow,
			expectReplyExists: true,
			expectScan:        true,
			expectCreate:      true,
			createErr:         errors.New("db down"),
			wantLabel:         "timeout_notice_failed",
		},
	}

	labels := []string{"timeout_notice_created", "timeout_notice_failed", "timeout_notice_skipped", "skipped_voice", "sent"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newPipelineErrorTestMocks(mc)

			sleeps := 0
			orig := backstopGraceSleep
			backstopGraceSleep = func(time.Duration) { sleeps++ }
			t.Cleanup(func() { backstopGraceSleep = orig })

			m.req.EXPECT().AIV1AIcallGet(gomock.Any(), testPEAicallID).Return(tt.cached, nil).Times(1)

			if tt.expectSleep {
				m.req.EXPECT().AIV1AIcallGetSkipCache(gomock.Any(), testPEAicallID).Return(tt.fresh, tt.freshErr).Times(1)
			} else {
				m.req.EXPECT().AIV1AIcallGetSkipCache(gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectLatest {
				m.db.EXPECT().MessageList(gomock.Any(), uint64(1), "", latestNonNotificationFilter).Return(tt.latest, tt.latestErr).Times(1)
			}
			if tt.expectReplyExists {
				m.db.EXPECT().MessageAssistantReplyExists(gomock.Any(), testPEPipecatcallID).Return(tt.replyExists, tt.replyExistsErr).Times(1)
			}
			if tt.expectScan {
				if tt.scanErr == nil {
					m.util.EXPECT().TimeNow().Return(&now)
				}
				m.db.EXPECT().MessageList(gomock.Any(), uint64(50), "", noticeScanFilter).Return(tt.scan, tt.scanErr).Times(1)
			}

			if tt.expectCreate {
				if tt.createErr != nil {
					m.util.EXPECT().UUIDCreate().Return(testPEMessageID)
					m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(tt.createErr).Times(1)
				} else {
					created := m.expectCreate(t)
					defer func() {
						if created.Role != message.RoleNotification || created.Direction != message.DirectionOutgoing {
							t.Errorf("Wrong role/direction. got: %s/%s", created.Role, created.Direction)
						}
						if created.AIcallID != testPEAicallID || created.CustomerID != testPECustomerID || created.ActiveflowID != testPEActiveflowID {
							t.Errorf("Wrong ids. got: %v", created)
						}
						if created.PipecatcallID != testPEPipecatcallID {
							t.Errorf("Wrong pipecatcall id. got: %s", created.PipecatcallID)
						}
						if created.ActiveAIID != testPEAIID {
							t.Errorf("Wrong active ai id. expect: %s, got: %s", testPEAIID, created.ActiveAIID)
						}
						expectContent := `{"type":"pipeline_error","category":"timeout","fatal":true,"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","message":"The AI language model did not respond in time."}`
						if created.Content != expectContent {
							t.Errorf("Wrong content.\nexpect: %s\ngot:    %s", expectContent, created.Content)
						}
					}()
				}
			} else {
				m.expectNoCreate()
			}

			before := map[string]float64{}
			for _, l := range labels {
				before[l] = testutil.ToFloat64(promBackstopReplyTotal.WithLabelValues(l))
			}

			if err := m.h.EventPMPipecatcallTerminated(context.Background(), newTerminatedEvent()); err != nil {
				t.Errorf("expected nil error, got %v", err)
			}

			if tt.expectSleep && sleeps != 1 {
				t.Errorf("expected exactly one grace sleep, got %d", sleeps)
			}
			if !tt.expectSleep && sleeps != 0 {
				t.Errorf("expected no grace sleep, got %d", sleeps)
			}

			for _, l := range labels {
				delta := testutil.ToFloat64(promBackstopReplyTotal.WithLabelValues(l)) - before[l]
				want := 0.0
				if l == tt.wantLabel {
					want = 1
				}
				if delta != want {
					t.Errorf("label %q: expected delta %v, got %v", l, want, delta)
				}
			}
		})
	}
}
