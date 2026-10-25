package subscribehandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
)

var errInjected = errors.New("injected handler failure")

// Test_processEvent_error_paths verifies, for every event processor, that a payload which is not
// valid JSON is rejected before any downstream handler is called, and that a downstream handler
// failure is returned to the caller with its cause intact.
func Test_processEvent_error_paths(t *testing.T) {
	validData := []byte(`{"id":"dd111111-0000-0000-0000-000000000001"}`)
	badData := []byte(`{"id":`)

	tests := []struct {
		name string
		call func(h *subscribeHandler, m *sock.Event) error

		// expectFail registers the single downstream call that must fail on valid data.
		expectFail func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler)
	}{
		{
			name: "call progressing",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCMCallProgressing(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventCMCallProgressing(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "call hangup",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCMCallHangup(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "customer deleted",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCMCustomerDeleted(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				ma.EXPECT().EventCUCustomerDeleted(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "customer created",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCMCustomerCreated(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				ma.EXPECT().EventCUCustomerCreated(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "customer frozen",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCUCustomerFrozen(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				ma.EXPECT().EventCUCustomerFrozen(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "customer recovered",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCUCustomerRecovered(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				ma.EXPECT().EventCUCustomerRecovered(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "email created",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventEMEmailCreated(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventEMEmailCreated(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "message created",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventMMMessageCreated(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventMMMessageCreated(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "number created",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventNMNumberCreated(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventNMNumberCreated(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "number renewed",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventNMNumberRenewed(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventNMNumberRenewed(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "recording started",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCMRecordingStarted(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventCMRecordingStarted(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "recording finished",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventCMRecordingFinished(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventCMRecordingFinished(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "tts speaking started",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventTTSSpeakingStarted(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventTTSSpeakingStarted(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
		{
			name: "tts speaking stopped",
			call: func(h *subscribeHandler, m *sock.Event) error {
				return h.processEventTTSSpeakingStopped(context.Background(), m)
			},
			expectFail: func(mb *billinghandler.MockBillingHandler, ma *accounthandler.MockAccountHandler) {
				mb.EXPECT().EventTTSSpeakingStopped(gomock.Any(), gomock.Any()).Return(errInjected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" invalid json", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation is registered on the mocks: any downstream call fails the test.
			h := &subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				billingHandler: billinghandler.NewMockBillingHandler(mc),
				accountHandler: accounthandler.NewMockAccountHandler(mc),
			}

			err := tt.call(h, &sock.Event{Data: badData})
			if err == nil {
				t.Fatalf("expected an unmarshal error, got nil")
			}
			if errors.Is(err, errInjected) || !strings.Contains(err.Error(), "could not unmarshal") {
				t.Errorf("expected the unmarshal failure, got: %v", err)
			}
		})

		t.Run(tt.name+" handler failure", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockBilling := billinghandler.NewMockBillingHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := &subscribeHandler{
				sockHandler:    sockhandler.NewMockSockHandler(mc),
				billingHandler: mockBilling,
				accountHandler: mockAccount,
			}
			tt.expectFail(mockBilling, mockAccount)

			err := tt.call(h, &sock.Event{Data: validData})
			if !errors.Is(err, errInjected) {
				t.Errorf("expected the injected cause, got: %v", err)
			}
		})
	}
}

// Test_Run_queue_create_failure verifies that Run stops at the first declaration failure.
func Test_Run_queue_create_failure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: "bin-manager.billing-manager.subscribe"}

	// no TopicCreateWithKind, QueueBind or ConsumeMessage expectation: they must not be reached.
	mockSock.EXPECT().QueueCreate(h.subscribeQueue, "normal").Return(errInjected)

	err := h.Run()
	if !errors.Is(err, errInjected) {
		t.Errorf("expected the injected cause, got: %v", err)
	}
}
