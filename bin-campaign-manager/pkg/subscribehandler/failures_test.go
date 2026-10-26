package subscribehandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
)

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockCampaign := campaignhandler.NewMockCampaignHandler(mc)
	mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)

	res := NewSubscribeHandler(mockSock, "test-subscribe-queue", mockCampaign, mockCampaigncall, nil)

	h, ok := res.(*subscribeHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *subscribeHandler, got: %T", res)
	}
	if h.sockHandler != mockSock || h.subscribeQueue != "test-subscribe-queue" ||
		h.campaignHandler != mockCampaign || h.campaigncallHandler != mockCampaigncall {
		t.Errorf("Wrong match. the given dependencies must be kept. handler: %+v", h)
	}
}

func Test_Run_setupFailures(t *testing.T) {
	queueName := "test-subscribe-queue"
	errBroker := stderrors.New("broker unavailable")

	tests := []struct {
		name  string
		setup func(m *sockhandler.MockSockHandler)

		expectMsg string
	}{
		{
			name: "queue declare fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(errBroker)
			},
			expectMsg: "could not declare the queue",
		},
		{
			name: "topic exchange declare fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(nil)
				m.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(errBroker)
			},
			expectMsg: "could not declare the global topic exchange",
		},
		{
			name: "topic binding fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(nil)
				m.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
				m.EXPECT().QueueBind(queueName, topicPatterns[0], string(commonoutline.QueueNameEvent), false, nil).Return(errBroker)
			},
			expectMsg: "could not bind the topic pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// ConsumeMessage has no expectation: a failed setup must not start consuming
			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: queueName}
			tt.setup(mockSock)

			err := h.Run()
			if err == nil || !stderrors.Is(err, errBroker) || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("Wrong match. expect: %s wrapping %v, got: %v", tt.expectMsg, errBroker, err)
			}
		})
	}
}

func Test_processEventFMActiveflowDeleted_branches(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("5e000001-0000-4000-8000-000000000001")
	campaignID := uuid.FromStringOrNil("5e000001-0000-4000-8000-000000000002")
	event := &sock.Event{
		Publisher: "flow-manager",
		Type:      "activeflow_deleted",
		DataType:  "application/json",
		Data:      []byte(`{"id":"` + activeflowID.String() + `"}`),
	}
	newCC := func(status campaigncall.Status, refType campaigncall.ReferenceType) *campaigncall.Campaigncall {
		return &campaigncall.Campaigncall{
			Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("5e000001-0000-4000-8000-000000000003")},
			CampaignID:    campaignID,
			Status:        status,
			ReferenceType: refType,
		}
	}
	errHandler := stderrors.New("handler failed")

	tests := []struct {
		name  string
		event *sock.Event
		setup func(mockCampaign *campaignhandler.MockCampaignHandler, mockCampaigncall *campaigncallhandler.MockCampaigncallHandler)

		expectErr bool
	}{
		{
			name:      "malformed data is an error and nothing is looked up",
			event:     &sock.Event{Publisher: "flow-manager", Type: "activeflow_deleted", Data: []byte(`{"broken`)},
			setup:     func(*campaignhandler.MockCampaignHandler, *campaigncallhandler.MockCampaigncallHandler) {},
			expectErr: true,
		},
		{
			name:  "unknown activeflow is dropped",
			event: event,
			setup: func(mockCampaign *campaignhandler.MockCampaignHandler, mockCampaigncall *campaigncallhandler.MockCampaigncallHandler) {
				mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), activeflowID).Return(nil, errHandler)
			},
		},
		{
			name:  "already done campaigncall is dropped",
			event: event,
			setup: func(mockCampaign *campaignhandler.MockCampaignHandler, mockCampaigncall *campaigncallhandler.MockCampaigncallHandler) {
				mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), activeflowID).Return(newCC(campaigncall.StatusDone, campaigncall.ReferenceTypeFlow), nil)
			},
		},
		{
			name:  "call reference is left to the call hangup handler",
			event: event,
			setup: func(mockCampaign *campaignhandler.MockCampaignHandler, mockCampaigncall *campaigncallhandler.MockCampaigncallHandler) {
				mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), activeflowID).Return(newCC(campaigncall.StatusProgressing, campaigncall.ReferenceTypeCall), nil)
			},
		},
		{
			name:  "campaigncall handler failure does not skip the campaign handler",
			event: event,
			setup: func(mockCampaign *campaignhandler.MockCampaignHandler, mockCampaigncall *campaigncallhandler.MockCampaigncallHandler) {
				cc := newCC(campaigncall.StatusProgressing, campaigncall.ReferenceTypeFlow)
				mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), activeflowID).Return(cc, nil)
				mockCampaigncall.EXPECT().EventHandleActiveflowDeleted(gomock.Any(), cc).Return(nil, errHandler)
				mockCampaign.EXPECT().EventHandleActiveflowDeleted(gomock.Any(), campaignID).Return(nil)
			},
		},
		{
			name:  "campaign handler failure is logged and not returned",
			event: event,
			setup: func(mockCampaign *campaignhandler.MockCampaignHandler, mockCampaigncall *campaigncallhandler.MockCampaigncallHandler) {
				cc := newCC(campaigncall.StatusProgressing, campaigncall.ReferenceTypeFlow)
				mockCampaigncall.EXPECT().GetByActiveflowID(gomock.Any(), activeflowID).Return(cc, nil)
				mockCampaigncall.EXPECT().EventHandleActiveflowDeleted(gomock.Any(), cc).Return(cc, nil)
				mockCampaign.EXPECT().EventHandleActiveflowDeleted(gomock.Any(), campaignID).Return(errHandler)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &subscribeHandler{campaignHandler: mockCampaign, campaigncallHandler: mockCampaigncall}
			tt.setup(mockCampaign, mockCampaigncall)

			err := h.processEventFMActiveflowDeleted(context.Background(), tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// captureLog collects the entries of the standard logger and restores its hooks on cleanup.
func captureLog(t *testing.T) *logrustest.Hook {
	t.Helper()

	hook := &logrustest.Hook{}
	saved := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.StandardLogger().AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(saved) })

	return hook
}

// Test_processEvent_ignoredAndFailed checks that an event nobody subscribes to is dropped
// without touching any handler, and that a failed handler is reported through the log.
func Test_processEvent_ignoredAndFailed(t *testing.T) {
	t.Run("unknown event is ignored", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// the handlers have no expectation, so any call to them fails the test
		h := &subscribeHandler{
			campaignHandler:     campaignhandler.NewMockCampaignHandler(mc),
			campaigncallHandler: campaigncallhandler.NewMockCampaigncallHandler(mc),
		}

		hook := captureLog(t)

		h.processEvent(&sock.Event{Publisher: "unknown-manager", Type: "unknown_event"})

		for _, entry := range hook.AllEntries() {
			if strings.Contains(entry.Message, "Could not process the event correctly") {
				t.Errorf("Wrong match. an ignored event must not be reported as failed. entry: %s", entry.Message)
			}
		}
	})

	t.Run("handler failure is logged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &subscribeHandler{
			campaignHandler:     campaignhandler.NewMockCampaignHandler(mc),
			campaigncallHandler: campaigncallhandler.NewMockCampaigncallHandler(mc),
		}

		hook := captureLog(t)

		h.processEvent(&sock.Event{
			Publisher: "flow-manager",
			Type:      "activeflow_deleted",
			Data:      []byte(`{"broken`),
		})

		found := false
		for _, entry := range hook.AllEntries() {
			if entry.Level == logrus.ErrorLevel && strings.Contains(entry.Message, "Could not process the event correctly") {
				found = true
			}
		}
		if !found {
			t.Errorf("Wrong match. expect: an error log for the failed event, got entries: %d", len(hook.AllEntries()))
		}
	})
}
