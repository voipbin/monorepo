package subscribehandler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/pkg/directhandler"
)

const testQueue = "bin-manager.direct-manager.subscribe.test"

// literalPattern is the one pattern direct-manager binds. It is a literal on purpose
// so that removing the entry from topicPatterns makes these tests fail.
const literalPattern = "customer-manager.customer.*.deleted"

func newCustomerDeletedEvent(t *testing.T, customerID uuid.UUID) *sock.Event {
	t.Helper()

	data, err := json.Marshal(&cmcustomer.Customer{ID: customerID})
	if err != nil {
		t.Fatalf("could not marshal the customer: %v", err)
	}

	return &sock.Event{
		Publisher: publisherCustomerManager,
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      json.RawMessage(data),
	}
}

func Test_Run_failures(t *testing.T) {
	errBroker := errors.New("broker failure")

	tests := []struct {
		name string

		queueCreateErr error
		topicErr       error
		bindErr        error

		expectBind  bool
		expectInErr string
		expectTopic bool
	}{
		{
			name:           "queue create fails",
			queueCreateErr: errBroker,
			expectInErr:    "could not declare the queue for subscribeHandler",
		},
		{
			name:        "topic exchange fails",
			topicErr:    errBroker,
			expectTopic: true,
			expectInErr: "could not declare the global topic exchange",
		},
		{
			name:        "bind fails",
			bindErr:     errBroker,
			expectTopic: true,
			expectBind:  true,
			expectInErr: "could not bind the topic pattern. pattern: " + literalPattern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			// the strict mock fails the test on any call that is not expected here, so a
			// failure at one step must stop the later steps and never start the consumer.
			mockSock.EXPECT().QueueCreate(testQueue, "normal").Return(tt.queueCreateErr)
			if tt.expectTopic {
				mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(tt.topicErr)
			}
			if tt.expectBind {
				mockSock.EXPECT().QueueBind(testQueue, literalPattern, string(commonoutline.QueueNameEvent), false, nil).Return(tt.bindErr)
			}

			h := NewSubscribeHandler(mockSock, testQueue, nil)

			err := h.Run()
			if err == nil {
				t.Fatalf("Run() error = nil, want an error containing %q", tt.expectInErr)
			}
			if !strings.Contains(err.Error(), tt.expectInErr) {
				t.Errorf("Run() error = %q, want it to contain %q", err.Error(), tt.expectInErr)
			}
			if !errors.Is(err, errBroker) {
				t.Errorf("Run() error = %v, want the broker error to be wrapped", err)
			}
		})
	}
}

func Test_Run_bindsLiteralPatternAndStartsConsumer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	consuming := make(chan struct{})
	gomock.InOrder(
		mockSock.EXPECT().QueueCreate(testQueue, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
		mockSock.EXPECT().QueueBind(testQueue, literalPattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil),
	)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), testQueue, string(commonoutline.ServiceNameDirectManager), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
			close(consuming)
			return nil
		}).Times(1)

	h := NewSubscribeHandler(mockSock, testQueue, nil)
	if err := h.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	select {
	case <-consuming:
	case <-time.After(5 * time.Second):
		t.Fatalf("the consumer was not started")
	}
}

func Test_Run_consumeErrorIsNotReturned(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	consuming := make(chan struct{})
	mockSock.EXPECT().QueueCreate(testQueue, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(testQueue, literalPattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), testQueue, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
			defer close(consuming)
			return errors.New("consume failure")
		}).Times(1)

	h := NewSubscribeHandler(mockSock, testQueue, nil)
	if err := h.Run(); err != nil {
		t.Fatalf("Run() error = %v, a consume failure happens in the background goroutine and is only logged", err)
	}

	select {
	case <-consuming:
	case <-time.After(5 * time.Second):
		t.Fatalf("the consumer was not started")
	}
}

func Test_processEvent(t *testing.T) {
	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")

	tests := []struct {
		name  string
		event *sock.Event

		expectCalls bool
		handlerErr  error
	}{
		{
			name:        "customer deleted is dispatched with the customer id",
			event:       newCustomerDeletedEvent(t, customerID),
			expectCalls: true,
		},
		{
			name:        "customer deleted handler error is only logged",
			event:       newCustomerDeletedEvent(t, customerID),
			expectCalls: true,
			handlerErr:  errors.New("handler failure"),
		},
		{
			name: "unknown publisher is ignored",
			event: &sock.Event{
				Publisher: "unknown-manager",
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      json.RawMessage(`{}`),
			},
		},
		{
			name: "unknown type of the customer manager is ignored",
			event: &sock.Event{
				Publisher: publisherCustomerManager,
				Type:      "customer_created",
				Data:      json.RawMessage(`{}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDirect := directhandler.NewMockDirectHandler(mc)
			if tt.expectCalls {
				mockDirect.EXPECT().EventCustomerDeleted(gomock.Any(), customerID).Return(tt.handlerErr).Times(1)
			}

			h := &subscribeHandler{directHandler: mockDirect}
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6")
	mockDirect := directhandler.NewMockDirectHandler(mc)

	handled := make(chan uuid.UUID, 1)
	mockDirect.EXPECT().EventCustomerDeleted(gomock.Any(), customerID).
		DoAndReturn(func(_ context.Context, id uuid.UUID) error {
			handled <- id
			return nil
		}).Times(1)

	h := &subscribeHandler{directHandler: mockDirect}

	// the event is processed in a goroutine, so the call itself must return nil at once.
	if err := h.processEventRun(newCustomerDeletedEvent(t, customerID)); err != nil {
		t.Fatalf("processEventRun() error = %v", err)
	}

	select {
	case got := <-handled:
		if got != customerID {
			t.Errorf("EventCustomerDeleted() got %v, want %v", got, customerID)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the event was not dispatched")
	}
}
