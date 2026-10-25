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

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/pkg/customerdomainhandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"
)

var errInjectedSubscribe = errors.New("injected failure")

// Test_Run_brokerFailures verifies that Run stops at the first failing synchronous
// broker step and does not go on to the later steps (the strict mock rejects them).
func Test_Run_brokerFailures(t *testing.T) {
	queueName := string(commonoutline.QueueNameRegistrarSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name  string
		setup func(m *sockhandler.MockSockHandler)

		expectErr string
	}{
		{
			name: "queue declaration fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(errInjectedSubscribe)
			},
			expectErr: "could not declare the queue",
		},
		{
			name: "topic exchange declaration fails",
			setup: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(errInjectedSubscribe),
				)
			},
			expectErr: "could not declare the global topic exchange",
		},
		{
			name: "the first binding fails",
			setup: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[0], exchange, false, nil).Return(errInjectedSubscribe),
				)
			},
			expectErr: "could not bind the topic pattern. pattern: " + topicPatterns[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.setup(mockSock)

			h := NewSubscribeHandler(mockSock, queueName, nil, nil, nil)

			err := h.Run()
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect an error containing %q, got: %v", tt.expectErr, err)
			}
			if !errors.Is(err, errInjectedSubscribe) {
				t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
			}
		})
	}
}

// Test_processEvent_handlerFailureIsLogged verifies that a failing handler is reported by
// processEvent and that the cascade stops at the failing handler.
func Test_processEvent_handlerFailureIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockTrunk := trunkhandler.NewMockTrunkHandler(mc)
	// the extension and customer domain handlers have no expectation: the cascade stops
	h := &subscribeHandler{
		extensionHandler:      extensionhandler.NewMockExtensionHandler(mc),
		trunkHandler:          mockTrunk,
		customerDomainHandler: customerdomainhandler.NewMockCustomerDomainHandler(mc),
	}

	oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	hook := logrustest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })

	mockTrunk.EXPECT().EventCUCustomerDeleted(gomock.Any(), gomock.Any()).Return(errInjectedSubscribe)

	h.processEvent(&sock.Event{
		Publisher: "customer-manager",
		Type:      cucustomer.EventTypeCustomerDeleted,
		DataType:  "application/json",
		Data:      []byte(`{"id":"4f8fbc3c-ccca-11ee-8104-9f5b184cb220"}`),
	})

	found := false
	for _, entry := range hook.AllEntries() {
		if entry.Data["func"] == "processEvent" && strings.Contains(entry.Message, "Could not process the event correctly") {
			found = true
			if !strings.Contains(entry.Message, string(cucustomer.EventTypeCustomerDeleted)) || !strings.Contains(entry.Message, errInjectedSubscribe.Error()) {
				t.Errorf("Wrong match. expect: the event type and the cause in the log line, got: %s", entry.Message)
			}
		}
	}
	if !found {
		t.Errorf("Wrong match. expect: the processing failure to be logged")
	}
}

// Test_processEventRun_dispatchesAsynchronously verifies that processEventRun hands the
// event to the handler without waiting for it: the handler is held until processEventRun
// has returned, and a synchronous dispatch would block it for the whole wait.
func Test_processEventRun_dispatchesAsynchronously(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCustomerDomain := customerdomainhandler.NewMockCustomerDomainHandler(mc)
	h := &subscribeHandler{customerDomainHandler: mockCustomerDomain}

	release := make(chan struct{})
	done := make(chan struct{})

	mockCustomerDomain.EXPECT().EventCUCustomerCreated(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *cucustomer.Customer) error {
		defer close(done)
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			t.Errorf("Wrong match. the handler was not released: the dispatch is not asynchronous")
		}
		return nil
	})

	err := h.processEventRun(&sock.Event{
		Publisher: "customer-manager",
		Type:      cucustomer.EventTypeCustomerCreated,
		DataType:  "application/json",
		Data:      []byte(`{"id":"9a1b0c2e-7f7f-11ee-8f5a-1f2e3d4c5b6a"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	close(release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. the handler was never called")
	}
}
