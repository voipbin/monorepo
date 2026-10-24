package subscribehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

type shMocks struct {
	sock      *sockhandler.MockSockHandler
	queue     *queuehandler.MockQueueHandler
	queuecall *queuecallhandler.MockQueuecallHandler
}

func newSH(mc *gomock.Controller) (*subscribeHandler, *shMocks) {
	m := &shMocks{
		sock:      sockhandler.NewMockSockHandler(mc),
		queue:     queuehandler.NewMockQueueHandler(mc),
		queuecall: queuecallhandler.NewMockQueuecallHandler(mc),
	}
	return &subscribeHandler{
		sockHandler:      m.sock,
		subscribeQueue:   "test-subscribe-queue",
		queueHandler:     m.queue,
		queuecallHandler: m.queuecall,
	}, m
}

func Test_processEventAMAgentStatusUpdated(t *testing.T) {
	agentID := uuid.FromStringOrNil("7d2e4b20-8f22-11ee-a101-000000000001")

	tests := []struct {
		name string

		data      string
		setup     func(m *shMocks)
		expectErr bool
	}{
		{
			name: "available agent is forwarded for matching",
			data: `{"id":"7d2e4b20-8f22-11ee-a101-000000000001","status":"available"}`,
			setup: func(m *shMocks) {
				m.queuecall.EXPECT().EventAMAgentAvailable(gomock.Any(), gomock.Cond(func(a amagent.Agent) bool {
					return a.ID == agentID && a.Status == amagent.StatusAvailable
				}))
			},
		},
		{
			name: "busy agent is ignored",
			data: `{"id":"7d2e4b20-8f22-11ee-a101-000000000001","status":"busy"}`,
		},
		{
			name: "offline agent is ignored",
			data: `{"id":"7d2e4b20-8f22-11ee-a101-000000000001","status":"offline"}`,
		},
		{
			name:      "malformed payload is an error and nothing is forwarded",
			data:      `not json`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newSH(mc)
			if tt.setup != nil {
				tt.setup(m)
			}

			err := h.processEventAMAgentStatusUpdated(context.Background(), &sock.Event{
				Publisher: string(commonoutline.ServiceNameAgentManager),
				Type:      amagent.EventTypeAgentStatusUpdated,
				Data:      []byte(tt.data),
			})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventCM_malformedPayload(t *testing.T) {
	tests := []struct {
		name string
		call func(h *subscribeHandler, m *sock.Event) error
	}{
		{"call hangup", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMCallHangup(context.Background(), m)
		}},
		{"confbridge joined", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMConfbridgeJoined(context.Background(), m)
		}},
		{"confbridge leaved", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMConfbridgeLeaved(context.Background(), m)
		}},
		{"customer deleted", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCUCustomerDeleted(context.Background(), m)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newSH(mc)

			// the mocks have no expectations: a malformed payload must not reach any handler.
			if err := tt.call(h, &sock.Event{Data: []byte(`not json`)}); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_processEventCMCallHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newSH(mc)

	callID := uuid.FromStringOrNil("7d2e4b20-8f22-11ee-a101-000000000002")
	m.queuecall.EXPECT().EventCallCallHangup(gomock.Any(), callID)

	err := h.processEventCMCallHangup(context.Background(), &sock.Event{
		Publisher: string(commonoutline.ServiceNameCallManager),
		Type:      cmcall.EventTypeCallHangup,
		Data:      []byte(`{"id":"7d2e4b20-8f22-11ee-a101-000000000002"}`),
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_processEventCUCustomerDeleted(t *testing.T) {
	errQueuecall := errors.New("queuecall cleanup failed")
	errQueue := errors.New("queue cleanup failed")
	customerID := uuid.FromStringOrNil("7d2e4b20-8f22-11ee-a101-000000000003")

	isCustomer := gomock.Cond(func(c *cucustomer.Customer) bool { return c != nil && c.ID == customerID })

	tests := []struct {
		name string

		setup     func(m *shMocks)
		expectErr error
	}{
		{
			name: "both handlers succeed",
			setup: func(m *shMocks) {
				m.queuecall.EXPECT().EventCUCustomerDeleted(gomock.Any(), isCustomer).Return(nil)
				m.queue.EXPECT().EventCUCustomerDeleted(gomock.Any(), isCustomer).Return(nil)
			},
		},
		{
			name: "queuecall cleanup failure stops before the queue cleanup",
			setup: func(m *shMocks) {
				m.queuecall.EXPECT().EventCUCustomerDeleted(gomock.Any(), isCustomer).Return(errQueuecall)
			},
			expectErr: errQueuecall,
		},
		{
			name: "queue cleanup failure is returned",
			setup: func(m *shMocks) {
				m.queuecall.EXPECT().EventCUCustomerDeleted(gomock.Any(), isCustomer).Return(nil)
				m.queue.EXPECT().EventCUCustomerDeleted(gomock.Any(), isCustomer).Return(errQueue)
			},
			expectErr: errQueue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newSH(mc)
			tt.setup(m)

			err := h.processEventCUCustomerDeleted(context.Background(), &sock.Event{
				Publisher: string(commonoutline.ServiceNameCustomerManager),
				Type:      cucustomer.EventTypeCustomerDeleted,
				Data:      []byte(`{"id":"7d2e4b20-8f22-11ee-a101-000000000003"}`),
			})
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEvent_routing(t *testing.T) {
	agentID := uuid.FromStringOrNil("7d2e4b20-8f22-11ee-a101-000000000001")

	t.Run("agent status updated event is routed to the agent handler", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSH(mc)
		m.queuecall.EXPECT().EventAMAgentAvailable(gomock.Any(), gomock.Cond(func(a amagent.Agent) bool { return a.ID == agentID }))

		h.processEvent(&sock.Event{
			Publisher: string(commonoutline.ServiceNameAgentManager),
			Type:      amagent.EventTypeAgentStatusUpdated,
			Data:      []byte(`{"id":"7d2e4b20-8f22-11ee-a101-000000000001","status":"available"}`),
		})
	})

	t.Run("call hangup event is routed to the call handler", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSH(mc)
		m.queuecall.EXPECT().EventCallCallHangup(gomock.Any(), agentID)

		h.processEvent(&sock.Event{
			Publisher: string(commonoutline.ServiceNameCallManager),
			Type:      cmcall.EventTypeCallHangup,
			Data:      []byte(`{"id":"7d2e4b20-8f22-11ee-a101-000000000001"}`),
		})
	})

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{"unknown publisher is ignored", &sock.Event{Publisher: "unknown-manager", Type: cmcall.EventTypeCallHangup, Data: []byte(`{}`)}},
		{"known publisher with an unhandled type is ignored", &sock.Event{Publisher: string(commonoutline.ServiceNameCallManager), Type: "call_created", Data: []byte(`{}`)}},
		{"agent publisher with another type is ignored", &sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: "agent_created", Data: []byte(`{}`)}},
		{"handler error is logged and swallowed", &sock.Event{Publisher: string(commonoutline.ServiceNameCallManager), Type: cmconfbridge.EventTypeConfbridgeJoined, Data: []byte(`not json`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _ := newSH(mc)

			// no expectations on the mocks: nothing may be forwarded.
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventRun_dispatchesAsynchronously(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newSH(mc)

	callID := uuid.FromStringOrNil("7d2e4b20-8f22-11ee-a101-000000000002")
	done := make(chan struct{})
	m.queuecall.EXPECT().EventCallCallHangup(gomock.Any(), callID).Do(func(_ any, _ uuid.UUID) {
		close(done)
	})

	err := h.processEventRun(&sock.Event{
		Publisher: string(commonoutline.ServiceNameCallManager),
		Type:      cmcall.EventTypeCallHangup,
		Data:      []byte(`{"id":"7d2e4b20-8f22-11ee-a101-000000000002"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("Wrong match. the event was not dispatched to the handler")
	}
}

func Test_Run_failures(t *testing.T) {
	queueName := "test-subscribe-queue"
	errBroker := errors.New("broker failure")

	t.Run("queue declaration failure stops before the topic is declared", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSH(mc)
		m.sock.EXPECT().QueueCreate(queueName, "normal").Return(errBroker)

		if err := h.Run(); !errors.Is(err, errBroker) {
			t.Errorf("Wrong match. expect: %v, got: %v", errBroker, err)
		}
	})

	t.Run("topic declaration failure stops before any binding", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSH(mc)
		m.sock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
		m.sock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(errBroker)

		if err := h.Run(); !errors.Is(err, errBroker) {
			t.Errorf("Wrong match. expect: %v, got: %v", errBroker, err)
		}
	})

	t.Run("binding failure on the first pattern stops the remaining bindings and the consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newSH(mc)
		m.sock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
		m.sock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
		m.sock.EXPECT().QueueBind(queueName, topicPatterns[0], string(commonoutline.QueueNameEvent), false, nil).Return(errBroker)

		if err := h.Run(); !errors.Is(err, errBroker) {
			t.Errorf("Wrong match. expect: %v, got: %v", errBroker, err)
		}
	})
}
