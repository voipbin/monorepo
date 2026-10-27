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

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
)

var errCov90 = errors.New("synthetic dependency failure")

func newCov90Handler(mc *gomock.Controller) (*subscribeHandler, *sockhandler.MockSockHandler, *flowhandler.MockFlowHandler, *activeflowhandler.MockActiveflowHandler) {
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockFlow := flowhandler.NewMockFlowHandler(mc)
	mockActive := activeflowhandler.NewMockActiveflowHandler(mc)

	h := &subscribeHandler{
		sockHandler:       mockSock,
		subscribeQueue:    "synthetic-subscribe-queue",
		flowHandler:       mockFlow,
		activeflowHandler: mockActive,
	}

	return h, mockSock, mockFlow, mockActive
}

func Test_Run_failure_coverage90(t *testing.T) {
	queue := "synthetic-subscribe-queue"
	pattern := topicPatterns[0]

	tests := []struct {
		name string

		setup func(mockSock *sockhandler.MockSockHandler)

		expectMessage string
	}{
		{
			name: "the queue can not be created",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				mockSock.EXPECT().QueueCreate(queue, "normal").Return(errCov90)
			},
			expectMessage: "could not declare the queue for subscribeHandler",
		},
		{
			name: "the topic exchange can not be created",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
					mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(errCov90),
				)
			},
			expectMessage: "could not declare the global topic exchange",
		},
		{
			name: "the topic pattern can not be bound",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
					mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
					mockSock.EXPECT().QueueBind(queue, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(errCov90),
				)
			},
			expectMessage: "could not bind the topic pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockSock, _, _ := newCov90Handler(mc)
			// the consume is never started when a step fails, so no ConsumeMessage is expected.
			tt.setup(mockSock)

			err := h.Run()
			if err == nil || !strings.Contains(err.Error(), tt.expectMessage) {
				t.Fatalf("Wrong error.\nexpect to contain: %s\ngot: %v", tt.expectMessage, err)
			}
			if !errors.Is(err, errCov90) {
				t.Errorf("The cause must be kept. got: %v", err)
			}
		})
	}
}

func Test_processEventCMCustomerDeleted_failure_coverage90(t *testing.T) {
	customerData := []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`)

	tests := []struct {
		name string

		data  []byte
		setup func(mockFlow *flowhandler.MockFlowHandler, mockActive *activeflowhandler.MockActiveflowHandler)

		expectMessage string
	}{
		{
			// no handler is called, because no expectation is registered.
			name:          "the data is not json",
			data:          []byte(`{"id":`),
			setup:         func(mockFlow *flowhandler.MockFlowHandler, mockActive *activeflowhandler.MockActiveflowHandler) {},
			expectMessage: "unexpected end of JSON input",
		},
		{
			// the activeflow handler is not called after the flow handler failed.
			name: "the flow handler fails",
			data: customerData,
			setup: func(mockFlow *flowhandler.MockFlowHandler, mockActive *activeflowhandler.MockActiveflowHandler) {
				mockFlow.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(errCov90)
			},
			expectMessage: "by the flow handler",
		},
		{
			name: "the activeflow handler fails",
			data: customerData,
			setup: func(mockFlow *flowhandler.MockFlowHandler, mockActive *activeflowhandler.MockActiveflowHandler) {
				mockFlow.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
				mockActive.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(errCov90)
			},
			expectMessage: "by the activeflow handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockFlow, mockActive := newCov90Handler(mc)
			tt.setup(mockFlow, mockActive)

			err := h.processEventCMCustomerDeleted(context.Background(), &sock.Event{
				Publisher: "customer-manager",
				Type:      cmcustomer.EventTypeCustomerDeleted,
				DataType:  "application/json",
				Data:      tt.data,
			})
			if err == nil || !strings.Contains(err.Error(), tt.expectMessage) {
				t.Errorf("Wrong error.\nexpect to contain: %s\ngot: %v", tt.expectMessage, err)
			}
		})
	}
}

func Test_processEvent_coverage90(t *testing.T) {
	t.Run("an event of an unknown publisher is ignored", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no expectation is registered, so a call into any handler fails the test.
		h, _, _, _ := newCov90Handler(mc)

		h.processEvent(&sock.Event{Publisher: "unknown-manager", Type: "unknown_event", Data: []byte(`{}`)})
	})

	t.Run("a failure of the handler is logged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockFlow, _ := newCov90Handler(mc)
		mockFlow.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(errCov90)

		hook := new(logrustest.Hook)
		old := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
		logrus.AddHook(hook)
		t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(old) })

		h.processEvent(&sock.Event{
			Publisher: "customer-manager",
			Type:      cmcustomer.EventTypeCustomerDeleted,
			DataType:  "application/json",
			Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
		})

		found := false
		for _, entry := range hook.AllEntries() {
			if entry.Level == logrus.ErrorLevel && strings.Contains(entry.Message, "Could not process the event correctly") {
				found = true
			}
		}
		if !found {
			t.Errorf("The failure of the event must be logged at the error level")
		}
	})
}

func Test_processEventRun_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockFlow, mockActive := newCov90Handler(mc)

	// the event is processed in the background, so the end of the processing is awaited.
	done := make(chan struct{})
	mockFlow.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
	mockActive.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *cmcustomer.Customer) error {
			defer close(done)
			return nil
		})

	err := h.processEventRun(&sock.Event{
		Publisher: "customer-manager",
		Type:      cmcustomer.EventTypeCustomerDeleted,
		DataType:  "application/json",
		Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("The event was not processed")
	}
}
