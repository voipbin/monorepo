package subscribehandler

import (
	"encoding/json"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_processEvent(t *testing.T) {
	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
	}

	customerData, _ := json.Marshal(customer)

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "processes_customer_deleted_event",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},
		},
		{
			name: "ignores_unknown_event",
			event: &sock.Event{
				Publisher: "unknown-service",
				Type:      "unknown_event",
				Data:      json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &subscribeHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			if tt.event.Publisher == publisherCustomerManager {
				mockTag.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
			}

			// processEvent runs in background, just call it directly
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "runs_process_event",
			event: &sock.Event{
				Publisher: "unknown-service",
				Type:      "unknown_event",
				Data:      json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			h := &subscribeHandler{
				sockHandler: mockSock,
				tagHandler:  mockTag,
			}

			err := h.processEventRun(tt.event)
			if err != nil {
				t.Errorf("processEventRun should not return error, got: %v", err)
			}
		})
	}
}

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	subscribeQueue := "test-queue"

	h := NewSubscribeHandler(mockSock, subscribeQueue, mockTag)

	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

// Test_Run_sequencing verifies the exact broker call sequence of Run() (VOIP-1407):
//
//	QueueCreate -> TopicCreateWithKind -> QueueBind (every topicPatterns entry, in order).
//
// The topic block MUST run synchronously inside Run(), before the ConsumeMessage
// goroutine: QueueBind and ConsumeMessage's internal basic.consume share the same AMQP
// channel, and racing them closes the channel with a 503 (production incident
// 2026-07-14, VOIP-1258). The strict gomock controller fails the test on any call outside
// the expected set; gomock.InOrder fails it on any reordering.
func Test_Run_sequencing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	queueName := string(commonoutline.QueueNameTagSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run(); it may or may not have been
	// scheduled by the time Run() returns, so it stays outside the InOrder chain.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameTagManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(
		mockSock,
		queueName,
		mockTag,
	)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}
