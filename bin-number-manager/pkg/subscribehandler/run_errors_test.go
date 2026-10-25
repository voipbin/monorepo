package subscribehandler

import (
	"errors"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
)

func Test_Run_errors(t *testing.T) {
	queueName := string(commonoutline.QueueNameNumberSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name         string
		setup        func(m *sockhandler.MockSockHandler, cause error)
		expectSubstr string
	}{
		{
			"queue create failure",
			func(m *sockhandler.MockSockHandler, cause error) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(cause)
			},
			"could not declare the queue",
		},
		{
			"topic exchange failure",
			func(m *sockhandler.MockSockHandler, cause error) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(cause),
				)
			},
			"could not declare the global topic exchange",
		},
		{
			// the second pattern fails: the first is bound, the failing pattern is reported and
			// nothing after it (no further bind, no consumer) is called.
			"bind failure on the last pattern",
			func(m *sockhandler.MockSockHandler, cause error) {
				calls := []any{
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
				}
				for i, pattern := range topicPatterns {
					var ret error
					if i == len(topicPatterns)-1 {
						ret = cause
					}
					calls = append(calls, m.EXPECT().QueueBind(queueName, pattern, exchange, false, nil).Return(ret))
				}
				gomock.InOrder(calls...)
			},
			"could not bind the topic pattern. pattern: " + topicPatterns[len(topicPatterns)-1],
		},
		{
			"bind failure on the first pattern",
			func(m *sockhandler.MockSockHandler, cause error) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[0], exchange, false, nil).Return(cause),
				)
			},
			"could not bind the topic pattern. pattern: " + topicPatterns[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			cause := errors.New("broker failure")
			tt.setup(mockSock, cause)

			h := NewSubscribeHandler(mockSock, queueName, numberhandler.NewMockNumberHandler(mc))

			err := h.Run()
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !errors.Is(err, cause) {
				t.Errorf("the broker error is not wrapped. got: %v", err)
			}
			if !strings.Contains(err.Error(), tt.expectSubstr) {
				t.Errorf("wrong message. expect it to contain %q, got: %v", tt.expectSubstr, err)
			}
		})
	}
}

func Test_Run_consumerFailureIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	queueName := string(commonoutline.QueueNameNumberSubscribe)
	mockSock := sockhandler.NewMockSockHandler(mc)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).Times(len(topicPatterns))

	started := make(chan struct{})
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "number-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ any) error {
			close(started)
			return errors.New("consume stopped")
		})

	h := NewSubscribeHandler(mockSock, queueName, numberhandler.NewMockNumberHandler(mc))
	if err := h.Run(); err != nil {
		t.Fatalf("a consumer failure must not fail Run. err: %v", err)
	}

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the consumer was not started")
	}
}

// Test_processEvent_ignored checks that an event no handler is registered for is dropped
// without calling the number handler. processEvent is called synchronously on purpose.
func Test_processEvent_ignored(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			"unknown publisher",
			&sock.Event{Publisher: "unknown-manager", Type: "customer_deleted", Data: []byte(`{}`)},
		},
		{
			"known publisher with an unrelated type",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: "customer_created", Data: []byte(`{}`)},
		},
		{
			"flow publisher with an unrelated type",
			&sock.Event{Publisher: publisherFlowManager, Type: "flow_created", Data: []byte(`{}`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the strict mock fails the test on any call.
			h := &subscribeHandler{numberHandler: numberhandler.NewMockNumberHandler(mc)}

			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun_returnsNil(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &subscribeHandler{numberHandler: numberhandler.NewMockNumberHandler(mc)}

	// an ignored event keeps the goroutine started by processEventRun free of mock calls.
	if err := h.processEventRun(&sock.Event{Publisher: "unknown-manager", Type: "x"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
