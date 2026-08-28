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

	"monorepo/bin-schedule-manager/pkg/schedulehandler"
)

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

	h := NewSubscribeHandler(mockSock, "bin-manager.schedule-manager.subscribe", []string{"bin-manager.customer-manager.event"}, mockSchedule)
	if h == nil {
		t.Errorf("Expected handler, got nil")
	}
}

// Test_Run_sequencing verifies the exact broker call sequence of Run() (VOIP-1406):
//
//	QueueCreate -> fanout QueueSubscribe (all subscribeTargets) -> TopicCreateWithKind
//	-> QueueBind (every topicPatterns entry, in order) -> QueueUnbind (every
//	fanoutUnbindTargets entry, in order).
//
// The topic block MUST run synchronously inside Run(), before the ConsumeMessage
// goroutine: QueueBind/QueueUnbind and ConsumeMessage's internal basic.consume share the
// same AMQP channel, and racing them closes the channel with a 503 (production incident
// 2026-07-14, VOIP-1258). The strict gomock controller fails the test on any call outside
// the expected set; gomock.InOrder fails it on any reordering.
func Test_Run_sequencing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

	queueName := string(commonoutline.QueueNameScheduleSubscribe)
	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	for _, target := range fanoutUnbindTargets {
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run(); it may or may not have been
	// scheduled by the time Run() returns, so it stays outside the InOrder chain.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameScheduleManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(
		mockSock,
		queueName,
		subscribeTargets,
		mockSchedule,
	)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

func Test_processEvent_routing(t *testing.T) {
	customer := &cmcustomer.Customer{
		ID: uuid.FromStringOrNil("29c7a2be-6f23-11f0-b3f6-9f2a3ba4bd12"),
	}
	customerData, _ := json.Marshal(customer)

	tests := []struct {
		name  string
		event *sock.Event

		expectHandled bool
	}{
		{
			name: "customer_deleted is handled",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},

			expectHandled: true,
		},
		{
			name: "other publisher is ignored",
			event: &sock.Event{
				Publisher: "call-manager",
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(customerData),
			},

			expectHandled: false,
		},
		{
			name: "other event type is ignored",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      "customer_created",
				Data:      json.RawMessage(customerData),
			},

			expectHandled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSchedule := schedulehandler.NewMockScheduleHandler(mc)

			h := &subscribeHandler{
				sockHandler:     mockSock,
				scheduleHandler: mockSchedule,
			}

			if tt.expectHandled {
				mockSchedule.EXPECT().EventCustomerDeleted(gomock.Any(), customer).Return(nil)
			}

			// call processEvent synchronously (processEventRun spawns it on a
			// goroutine in production; the routing logic is what is under test)
			h.processEvent(tt.event)
		})
	}
}
