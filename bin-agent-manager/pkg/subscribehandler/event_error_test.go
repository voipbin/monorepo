package subscribehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processEvent_handlerErrorsAreWrapped(t *testing.T) {
	errBoom := stderrors.New("boom")

	tests := []struct {
		name string

		event    *sock.Event
		mockFunc func(m *agenthandler.MockAgentHandler)
		call     func(h *subscribeHandler, ctx context.Context, e *sock.Event) error

		expectErr string
	}{
		{
			name: "groupcall progressing",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCallManager),
				Type:      string(cmgroupcall.EventTypeGroupcallProgressing),
				Data:      []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","customer_id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventGroupcallProgressing(gomock.Any(), gomock.Any()).Return(errBoom)
			},
			call: func(h *subscribeHandler, ctx context.Context, e *sock.Event) error {
				return h.processEventCMGroupcallProgressing(ctx, e)
			},

			expectErr: "Could not handle the groupcall answered event.: boom",
		},
		{
			name: "customer deleted",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      []byte(`{"id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(errBoom)
			},
			call: func(h *subscribeHandler, ctx context.Context, e *sock.Event) error {
				return h.processEventCMCustomerDeleted(ctx, e)
			},

			expectErr: "Could not handle the customer deleted event.: boom",
		},
		{
			name: "customer created",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerCreated),
				Data:      []byte(`{"id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc","email":"owner@example.com","headless":true}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerCreated(gomock.Any(), gomock.Any(), true).Return(errBoom)
			},
			call: func(h *subscribeHandler, ctx context.Context, e *sock.Event) error {
				return h.processEventCMCustomerCreated(ctx, e)
			},

			expectErr: "Could not handle the customer created event.: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &subscribeHandler{agentHandler: mockAgent}

			tt.mockFunc(mockAgent)

			err := tt.call(h, context.Background(), tt.event)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong error.\nexpect: %s\ngot: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEvent_ignoredAndFailedEvents(t *testing.T) {
	tests := []struct {
		name string

		event    *sock.Event
		mockFunc func(m *agenthandler.MockAgentHandler)
	}{
		{
			name: "unknown publisher is ignored without calling the agent handler",

			event: &sock.Event{
				Publisher: "unknown-manager",
				Type:      "unknown_event",
				Data:      []byte(`{}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {},
		},
		{
			name: "handler failure is consumed after exactly one handler call",

			event: &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      string(cmcustomer.EventTypeCustomerDeleted),
				Data:      []byte(`{"id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc"}`),
			},
			mockFunc: func(m *agenthandler.MockAgentHandler) {
				m.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(stderrors.New("boom")).Times(1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &subscribeHandler{agentHandler: mockAgent}

			tt.mockFunc(mockAgent)

			h.processEvent(tt.event)
		})
	}
}

// Test_processEventRun_doesNotWaitForHandler verifies that processEventRun hands the event over
// to a separate goroutine and returns before the handler finishes.
func Test_processEventRun_doesNotWaitForHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAgent := agenthandler.NewMockAgentHandler(mc)
	h := &subscribeHandler{agentHandler: mockAgent}

	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	defer close(release)

	mockAgent.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ *cmcustomer.Customer) error {
			close(entered)
			<-release
			close(finished)
			return nil
		}).Times(1)

	event := &sock.Event{
		Publisher: string(commonoutline.ServiceNameCustomerManager),
		Type:      string(cmcustomer.EventTypeCustomerDeleted),
		Data:      []byte(`{"id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc"}`),
	}

	returned := make(chan error, 1)
	go func() {
		returned <- h.processEventRun(event)
	}()

	select {
	case err := <-returned:
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("processEventRun did not return while the handler was blocked.")
		return
	}

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Errorf("The handler was not called.")
		return
	}

	select {
	case <-finished:
		t.Errorf("The handler finished before it was released.")
	default:
	}
}
