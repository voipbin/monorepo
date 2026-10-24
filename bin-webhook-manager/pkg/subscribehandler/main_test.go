package subscribehandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
	"monorepo/bin-webhook-manager/models/webhook"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// Test_Run_TopicMigrationSequencing asserts the full VOIP-1407 broker call sequence inside
// Run(), in strict order: QueueCreate -> TopicCreateWithKind -> each topic QueueBind (all
// patterns). The whole sequence MUST complete synchronously before the ConsumeMessage
// goroutine starts: QueueBind and basic.consume share one AMQP channel, and racing them
// closes the channel with a 503 (VOIP-1258 production incident, 2026-07-14).
func Test_Run_TopicMigrationSequencing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	queueName := string(commonoutline.QueueNameWebhookSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run() -- it may or may not have been
	// scheduled by the time Run() returns, so expect it loosely outside the ordered chain
	// (matching the bin-agent-manager Run() test idiom).
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "webhook-manager", false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, mockAccount, mockCache)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

func Test_Run_errors(t *testing.T) {
	queueName := string(commonoutline.QueueNameWebhookSubscribe)
	eventExchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name string

		setup func(mockSock *sockhandler.MockSockHandler)

		expectErrContains string
	}{
		{
			name: "queue create failure stops before any exchange call",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				mockSock.EXPECT().QueueCreate(queueName, "normal").Return(errors.New("boom"))
			},
			expectErrContains: "could not declare the queue",
		},
		{
			name: "exchange declare failure stops before any bind",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					mockSock.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(errors.New("boom")),
				)
			},
			expectErrContains: "could not declare the global topic exchange",
		},
		{
			name: "bind failure stops before consuming",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					mockSock.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(nil),
					mockSock.EXPECT().QueueBind(queueName, "customer-manager.customer.*.created", eventExchange, false, nil).Return(errors.New("boom")),
				)
			},
			expectErrContains: "could not bind the topic pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// unexpected calls (further binds, ConsumeMessage) fail the test.
			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.setup(mockSock)

			h := NewSubscribeHandler(mockSock, queueName, accounthandler.NewMockAccountHandler(mc), cachehandler.NewMockCacheHandler(mc))

			err := h.Run()
			if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErrContains, err)
			}
		})
	}
}

func Test_Run_consumeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	queueName := string(commonoutline.QueueNameWebhookSubscribe)
	mockSock := sockhandler.NewMockSockHandler(mc)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range []string{
		"customer-manager.customer.*.created",
		"customer-manager.customer.*.updated",
		"flow-manager.activeflow.*.created",
		"flow-manager.activeflow.*.updated",
		"flow-manager.activeflow.*.deleted",
	} {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// the consume loop runs in a goroutine; its failure must not change Run's result.
	consumed := make(chan struct{})
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "webhook-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgConsume) error {
			close(consumed)
			return errors.New("consume failed")
		}).Times(1)

	h := NewSubscribeHandler(mockSock, queueName, accounthandler.NewMockAccountHandler(mc), cachehandler.NewMockCacheHandler(mc))
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Fatalf("ConsumeMessage was not called")
	}
}

func Test_processEvent_flowManagerRouting(t *testing.T) {
	tmCreate := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC)
	activeflowID := uuid.FromStringOrNil("e1111111-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		event *sock.Event

		setup func(mockCache *cachehandler.MockCacheHandler)
	}{
		{
			name: "created with uri writes a positive entry",
			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"e1111111-1111-1111-1111-111111111111","webhook_uri":"https://example.com/hook","webhook_method":"POST","tm_create":"2026-06-10T00:00:00Z"}`),
			},
			setup: func(mockCache *cachehandler.MockCacheHandler) {
				mockCache.EXPECT().ActiveflowWebhookSet(gomock.Any(), activeflowID, &mwactiveflow.Webhook{
					URI:    "https://example.com/hook",
					Method: webhook.MethodTypePOST,
					Tm:     tmCreate,
				}, subTLive).Return(nil).Times(1)
			},
		},
		{
			name: "updated without uri writes a negative entry",
			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowUpdated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"e1111111-1111-1111-1111-111111111111","tm_create":"2026-06-10T00:00:00Z"}`),
			},
			setup: func(mockCache *cachehandler.MockCacheHandler) {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), activeflowID, tmCreate, nil, subTNeg).Return(nil).Times(1)
			},
		},
		{
			name: "deleted writes a tombstone carrying the delete time",
			event: &sock.Event{
				Publisher: publisherFlowManager,
				Type:      fmactiveflow.EventTypeActiveflowDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"id":"e1111111-1111-1111-1111-111111111111","tm_create":"2026-06-10T00:00:00Z","tm_delete":"2026-06-10T01:00:00Z"}`),
			},
			setup: func(mockCache *cachehandler.MockCacheHandler) {
				mockCache.EXPECT().ActiveflowWebhookSetNegative(gomock.Any(), activeflowID, tmDelete, &tmDelete, subTNeg).Return(nil).Times(1)
			},
		},
		{
			name: "activeflow type from another publisher is ignored",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      fmactiveflow.EventTypeActiveflowCreated,
				DataType:  "application/json",
				Data:      []byte(`{"id":"e1111111-1111-1111-1111-111111111111"}`),
			},
			setup: func(mockCache *cachehandler.MockCacheHandler) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			if activeflowID == uuid.Nil {
				t.Fatalf("invalid test uuid")
			}

			mockCache := cachehandler.NewMockCacheHandler(mc)
			tt.setup(mockCache)

			h := &subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				accountHandler: accounthandler.NewMockAccountHandler(mc),
				cacheHandler:   mockCache,
			}

			h.processEvent(tt.event)
		})
	}
}
