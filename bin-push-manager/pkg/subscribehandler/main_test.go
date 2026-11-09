package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/pkg/devicehandler"
)

func Test_processEvent(t *testing.T) {
	agentID := uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	customerID := uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")

	agentData, _ := json.Marshal(&amagent.Agent{Identity: commonidentity.Identity{ID: agentID}})
	customerData, _ := json.Marshal(&cmcustomer.Customer{ID: customerID})

	tests := []struct {
		name   string
		event  *sock.Event
		expect func(m *devicehandler.MockDeviceHandler, done chan struct{})
	}{
		{
			"agent_deleted deletes the devices of the agent",
			&sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: amagent.EventTypeAgentDeleted, DataType: "application/json", Data: agentData},
			func(m *devicehandler.MockDeviceHandler, done chan struct{}) {
				m.EXPECT().EventAgentDeleted(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *amagent.Agent) error {
					defer close(done)
					if a.ID != agentID {
						t.Errorf("Wrong match. expect: %s, got: %s", agentID, a.ID)
					}
					return nil
				})
			},
		},
		{
			"customer_deleted deletes the devices of the customer",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: cmcustomer.EventTypeCustomerDeleted, DataType: "application/json", Data: customerData},
			func(m *devicehandler.MockDeviceHandler, done chan struct{}) {
				m.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c *cmcustomer.Customer) error {
					defer close(done)
					if c.ID != customerID {
						t.Errorf("Wrong match. expect: %s, got: %s", customerID, c.ID)
					}
					return nil
				})
			},
		},
		{
			"the handler error is logged, not raised",
			&sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: amagent.EventTypeAgentDeleted, DataType: "application/json", Data: agentData},
			func(m *devicehandler.MockDeviceHandler, done chan struct{}) {
				m.EXPECT().EventAgentDeleted(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *amagent.Agent) error {
					defer close(done)
					return fmt.Errorf("boom")
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDevice := devicehandler.NewMockDeviceHandler(mc)
			h := &subscribeHandler{deviceHandler: mockDevice}

			done := make(chan struct{})
			tt.expect(mockDevice, done)

			if err := h.processEventRun(tt.event); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("Wrong match. the handler was not called")
			}
		})
	}
}

func Test_processEvent_ignored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no handler call is expected for another publisher, another type, or a malformed body
	h := &subscribeHandler{deviceHandler: devicehandler.NewMockDeviceHandler(mc)}

	h.processEvent(&sock.Event{Publisher: "call-manager", Type: amagent.EventTypeAgentDeleted})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: "agent_created"})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: amagent.EventTypeAgentDeleted, Data: []byte("not json")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: cmcustomer.EventTypeCustomerDeleted, Data: []byte("not json")})
}

func Test_Run(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(m *sockhandler.MockSockHandler)
		expectErr bool
	}{
		{"queue create fails", func(m *sockhandler.MockSockHandler) {
			m.EXPECT().QueueCreate("q", "normal").Return(fmt.Errorf("boom"))
		}, true},
		{"topic create fails", func(m *sockhandler.MockSockHandler) {
			m.EXPECT().QueueCreate("q", "normal").Return(nil)
			m.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("boom"))
		}, true},
		{"bind fails", func(m *sockhandler.MockSockHandler) {
			m.EXPECT().QueueCreate("q", "normal").Return(nil)
			m.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
			m.EXPECT().QueueBind("q", gomock.Any(), string(commonoutline.QueueNameEvent), false, gomock.Any()).Return(fmt.Errorf("boom"))
		}, true},
		{"all patterns are bound and the consumer starts", func(m *sockhandler.MockSockHandler) {
			m.EXPECT().QueueCreate("q", "normal").Return(nil)
			m.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
			m.EXPECT().QueueBind("q", gomock.Any(), string(commonoutline.QueueNameEvent), false, gomock.Any()).Return(nil).Times(len(topicPatterns))
			m.EXPECT().ConsumeMessage(gomock.Any(), "q", string(commonoutline.ServiceNamePushManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.setup(mockSock)

			h := NewSubscribeHandler(mockSock, "q", devicehandler.NewMockDeviceHandler(mc))
			if err := h.Run(); (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			// let the consumer goroutine reach its mock call
			time.Sleep(20 * time.Millisecond)
		})
	}
}
