package messagehandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
)

var (
	testPEAicallID        = uuid.FromStringOrNil("443bfb46-fa3a-4dd5-912d-5482d31fed22")
	testPECustomerID      = uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b")
	testPEActiveflowID    = uuid.FromStringOrNil("8903319a-ddcd-4283-819e-bd7f6ce8c3f0")
	testPEAIID            = uuid.FromStringOrNil("6e391666-9078-42a1-8a1c-7e07abfb48ec")
	testPEPipecatcallID   = uuid.FromStringOrNil("9c5c6e64-6289-4ac9-ad88-b20796a6bc96")
	testPEOtherPipecallID = uuid.FromStringOrNil("0d26685f-18bd-4b85-8b03-eec4265326f8")
	testPEMessageID       = uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000001")
)

func newPipelineErrorEvent(category pmmessage.ErrorCategory) *pmmessage.PipelineErrorEvent {
	return &pmmessage.PipelineErrorEvent{
		CustomerID:               testPECustomerID,
		PipecatcallID:            testPEPipecatcallID,
		PipecatcallReferenceType: pmpipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   testPEAicallID,
		ActiveflowID:             testPEActiveflowID,
		Category:                 category,
	}
}

func newPipelineErrorAIcall(currentPipecatcallID uuid.UUID) *aicall.AIcall {
	ac := &aicall.AIcall{
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   testPEAIID,
		PipecatcallID:  currentPipecatcallID,
	}
	ac.ID = testPEAicallID
	return ac
}

type pipelineErrorTestMocks struct {
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	util   *utilhandler.MockUtilHandler
	notify *notifyhandler.MockNotifyHandler
	h      *messageHandler
}

func newPipelineErrorTestMocks(mc *gomock.Controller) *pipelineErrorTestMocks {
	m := &pipelineErrorTestMocks{
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
	}
	m.h = &messageHandler{
		db:            m.db,
		reqHandler:    m.req,
		utilHandler:   m.util,
		notifyHandler: m.notify,
	}
	return m
}

// expectCreate expects exactly one notice row and returns a pointer that receives it.
func (m *pipelineErrorTestMocks) expectCreate(t *testing.T) *message.Message {
	created := &message.Message{}
	m.util.EXPECT().UUIDCreate().Return(testPEMessageID)
	m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, msg *message.Message) error {
			*created = *msg
			return nil
		},
	).Times(1)
	m.db.EXPECT().MessageGet(gomock.Any(), testPEMessageID).DoAndReturn(
		func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return created, nil
		},
	).Times(1)
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), testPECustomerID, message.EventTypeMessageCreated, gomock.Any()).Times(1)
	return created
}

func (m *pipelineErrorTestMocks) expectNoCreate() {
	m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Times(0)
}

func noticeRow(t *testing.T, category pmmessage.ErrorCategory, created time.Time) *message.Message {
	content, err := json.Marshal(PipelineErrorNotice{Type: NotificationTypePipelineError, Category: category, PipecatcallID: testPEOtherPipecallID})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	res := &message.Message{Role: message.RoleNotification, Content: string(content), TMCreate: &created}
	return res
}

func Test_EventPMPipelineError_currentPipecatcall(t *testing.T) {
	tests := []struct {
		name          string
		category      pmmessage.ErrorCategory
		fatal         bool
		expectContent string
	}{
		{
			name:          "authentication",
			category:      pmmessage.ErrorCategoryAuthentication,
			expectContent: `{"type":"pipeline_error","category":"authentication","fatal":false,"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","message":"An AI service provider rejected the credentials or denied access. If this AI uses a custom engine key, verify that the key is valid and permitted for the selected model."}`,
		},
		{
			name:          "rate limited",
			category:      pmmessage.ErrorCategoryRateLimited,
			expectContent: `{"type":"pipeline_error","category":"rate_limited","fatal":false,"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","message":"An AI service provider rejected the request due to a rate limit or quota. Try again later."}`,
		},
		{
			name:          "timeout",
			category:      pmmessage.ErrorCategoryTimeout,
			expectContent: `{"type":"pipeline_error","category":"timeout","fatal":false,"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","message":"The AI language model did not respond in time."}`,
		},
		{
			name:          "unknown fatal",
			category:      pmmessage.ErrorCategoryUnknown,
			fatal:         true,
			expectContent: `{"type":"pipeline_error","category":"unknown","fatal":true,"pipecatcall_id":"9c5c6e64-6289-4ac9-ad88-b20796a6bc96","message":"An AI service provider returned an error."}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newPipelineErrorTestMocks(mc)

			evt := newPipelineErrorEvent(tt.category)
			evt.Fatal = tt.fatal

			// current pipecatcall: no dedup lookup at all
			m.req.EXPECT().AIV1AIcallGet(gomock.Any(), testPEAicallID).Return(newPipelineErrorAIcall(testPEPipecatcallID), nil).Times(1)
			m.req.EXPECT().AIV1AIcallGetSkipCache(gomock.Any(), gomock.Any()).Times(0)
			m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			created := m.expectCreate(t)

			m.h.EventPMPipelineError(context.Background(), evt)

			if created.Content != tt.expectContent {
				t.Errorf("Wrong match.\nexpect: %s\ngot:    %s", tt.expectContent, created.Content)
			}
			if created.Role != message.RoleNotification || created.Direction != message.DirectionOutgoing {
				t.Errorf("Wrong role/direction. got: %s/%s", created.Role, created.Direction)
			}
			if created.AIcallID != testPEAicallID || created.ActiveflowID != testPEActiveflowID || created.CustomerID != testPECustomerID {
				t.Errorf("Wrong ids. got: %v", created)
			}
			if created.PipecatcallID != testPEPipecatcallID {
				t.Errorf("Wrong pipecatcall id. expect: %s, got: %s", testPEPipecatcallID, created.PipecatcallID)
			}
			if created.ActiveAIID != testPEAIID {
				t.Errorf("Wrong active ai id. expect: %s, got: %s", testPEAIID, created.ActiveAIID)
			}
		})
	}
}

func Test_EventPMPipelineError_currentPipecatcallNotWindowed(t *testing.T) {
	// A current-pipecatcall event is never windowed, even when a same-category notice was just
	// created (the operator retrying after editing the key must see whether it failed again).
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newPipelineErrorTestMocks(mc)

	m.req.EXPECT().AIV1AIcallGet(gomock.Any(), testPEAicallID).Return(newPipelineErrorAIcall(testPEPipecatcallID), nil)
	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.expectCreate(t)

	m.h.EventPMPipelineError(context.Background(), newPipelineErrorEvent(pmmessage.ErrorCategoryAuthentication))
}

func Test_EventPMPipelineError_nonAIcall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newPipelineErrorTestMocks(mc)

	evt := newPipelineErrorEvent(pmmessage.ErrorCategoryAuthentication)
	evt.PipecatcallReferenceType = pmpipecatcall.ReferenceTypeCall

	m.req.EXPECT().AIV1AIcallGet(gomock.Any(), gomock.Any()).Times(0)
	m.expectNoCreate()

	m.h.EventPMPipelineError(context.Background(), evt)
}

func Test_EventPMPipelineError_aicallGetFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newPipelineErrorTestMocks(mc)

	m.req.EXPECT().AIV1AIcallGet(gomock.Any(), testPEAicallID).Return(nil, errors.New("boom"))
	m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	created := m.expectCreate(t)

	m.h.EventPMPipelineError(context.Background(), newPipelineErrorEvent(pmmessage.ErrorCategoryAuthentication))

	if created.ActiveAIID != uuid.Nil {
		t.Errorf("Wrong active ai id. expect: nil, got: %s", created.ActiveAIID)
	}
}

func Test_EventPMPipelineError_createFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newPipelineErrorTestMocks(mc)

	m.req.EXPECT().AIV1AIcallGet(gomock.Any(), testPEAicallID).Return(newPipelineErrorAIcall(testPEPipecatcallID), nil)
	m.util.EXPECT().UUIDCreate().Return(testPEMessageID)
	m.db.EXPECT().MessageCreate(gomock.Any(), gomock.Any()).Return(errors.New("db down"))
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// must not panic
	m.h.EventPMPipelineError(context.Background(), newPipelineErrorEvent(pmmessage.ErrorCategoryAuthentication))
}

func Test_EventPMPipelineError_foreignPipecatcall(t *testing.T) {
	now := time.Date(2026, 9, 29, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		freshPipecatcallID uuid.UUID // what the SkipCache read returns
		skipCacheErr       error

		expectList bool
		listRes    []*message.Message
		listErr    error

		expectCreate bool
	}{
		{
			name:               "same category within window is skipped",
			freshPipecatcallID: testPEOtherPipecallID,
			expectList:         true,
			listRes:            []*message.Message{noticeRow(t, pmmessage.ErrorCategoryAuthentication, now.Add(-5*time.Minute))},
			expectCreate:       false,
		},
		{
			name:               "same category outside window creates",
			freshPipecatcallID: testPEOtherPipecallID,
			expectList:         true,
			listRes:            []*message.Message{noticeRow(t, pmmessage.ErrorCategoryAuthentication, now.Add(-11*time.Minute))},
			expectCreate:       true,
		},
		{
			name:               "different category within window creates",
			freshPipecatcallID: testPEOtherPipecallID,
			expectList:         true,
			listRes:            []*message.Message{noticeRow(t, pmmessage.ErrorCategoryTimeout, now.Add(-1*time.Minute))},
			expectCreate:       true,
		},
		{
			name:               "no prior notice creates (stale superseded turn)",
			freshPipecatcallID: testPEOtherPipecallID,
			expectList:         true,
			listRes:            []*message.Message{},
			expectCreate:       true,
		},
		{
			name:               "unparseable and non pipeline_error rows are ignored",
			freshPipecatcallID: testPEOtherPipecallID,
			expectList:         true,
			listRes: []*message.Message{
				{Role: message.RoleNotification, Content: "not-json", TMCreate: &now},
				{Role: message.RoleNotification, Content: `{"type":"member_switched","category":"authentication"}`, TMCreate: &now},
			},
			expectCreate: true,
		},
		{
			name:               "list failure creates (fail-open)",
			freshPipecatcallID: testPEOtherPipecallID,
			expectList:         true,
			listErr:            errors.New("db down"),
			expectCreate:       true,
		},
		{
			name:               "stale cache: fresh read says current, not windowed",
			freshPipecatcallID: testPEPipecatcallID,
			expectList:         false,
			expectCreate:       true,
		},
		{
			name:         "skip-cache failure treated as current, not windowed",
			skipCacheErr: errors.New("db down"),
			expectList:   false,
			expectCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newPipelineErrorTestMocks(mc)

			// cached row says a different pipecatcall is current -> foreign
			m.req.EXPECT().AIV1AIcallGet(gomock.Any(), testPEAicallID).Return(newPipelineErrorAIcall(testPEOtherPipecallID), nil)
			if tt.skipCacheErr != nil {
				m.req.EXPECT().AIV1AIcallGetSkipCache(gomock.Any(), testPEAicallID).Return(nil, tt.skipCacheErr)
			} else {
				m.req.EXPECT().AIV1AIcallGetSkipCache(gomock.Any(), testPEAicallID).Return(newPipelineErrorAIcall(tt.freshPipecatcallID), nil)
			}

			if tt.expectList {
				if tt.listErr == nil {
					m.util.EXPECT().TimeNow().Return(&now)
				}
				m.db.EXPECT().MessageList(gomock.Any(), uint64(pipelineErrorNoticeScanSize), "", map[message.Field]any{
					message.FieldAIcallID: testPEAicallID,
					message.FieldRole:     message.RoleNotification,
					message.FieldDeleted:  false,
				}).Return(tt.listRes, tt.listErr)
			} else {
				m.db.EXPECT().MessageList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectCreate {
				created := m.expectCreate(t)
				defer func() {
					if created.PipecatcallID != testPEPipecatcallID {
						t.Errorf("Wrong pipecatcall id. got: %s", created.PipecatcallID)
					}
				}()
			} else {
				m.expectNoCreate()
			}

			m.h.EventPMPipelineError(context.Background(), newPipelineErrorEvent(pmmessage.ErrorCategoryAuthentication))
		})
	}
}
