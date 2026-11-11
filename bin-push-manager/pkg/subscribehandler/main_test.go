package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	tkmessage "monorepo/bin-talk-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-push-manager/pkg/devicehandler"
	"monorepo/bin-push-manager/pkg/eventhandler"
	"monorepo/bin-push-manager/pkg/settingshandler"
)

var (
	testAgentID    = uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	testCustomerID = uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	testChatID     = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e11")
	testMessageID  = uuid.FromStringOrNil("5d7f9b1c-3e4a-4c68-8a0e-2c4e6a8c0e12")
	testConvID     = uuid.FromStringOrNil("6e8a0c2d-4f5b-4d79-9b1f-3d5f7b9d1f21")
	testCaseID     = uuid.FromStringOrNil("7f9b1d3e-5a6c-4e8a-8c2a-4e6a8c0e2a31")
	testNoteID     = uuid.FromStringOrNil("7f9b1d3e-5a6c-4e8a-8c2a-4e6a8c0e2a32")
)

type testMocks struct {
	device   *devicehandler.MockDeviceHandler
	settings *settingshandler.MockSettingsHandler
	event    *eventhandler.MockEventHandler
}

// newTestHandler builds a handler with strict mocks and a pool of semSize slots.
func newTestHandler(mc *gomock.Controller, semSize int) (*subscribeHandler, *testMocks) {
	m := &testMocks{
		device:   devicehandler.NewMockDeviceHandler(mc),
		settings: settingshandler.NewMockSettingsHandler(mc),
		event:    eventhandler.NewMockEventHandler(mc),
	}

	return &subscribeHandler{
		deviceHandler:   m.device,
		settingsHandler: m.settings,
		eventHandler:    m.event,
		sem:             make(chan struct{}, semSize),
	}, m
}

// waitIdle returns when every started event is over: it takes all the slots of the pool.
func waitIdle(t *testing.T, h *subscribeHandler) {
	t.Helper()

	for i := 0; i < cap(h.sem); i++ {
		select {
		case h.sem <- struct{}{}:
		case <-time.After(2 * time.Second):
			t.Fatalf("Wrong match. an event did not finish")
		}
	}
	for i := 0; i < cap(h.sem); i++ {
		<-h.sem
	}
}

func event(publisher commonoutline.ServiceName, eventType string, data any) *sock.Event {
	m, _ := json.Marshal(data)
	return &sock.Event{Publisher: string(publisher), Type: eventType, DataType: "application/json", Data: m}
}

func Test_processEvent_deleted(t *testing.T) {
	agentEvent := event(commonoutline.ServiceNameAgentManager, amagent.EventTypeAgentDeleted, &amagent.Agent{Identity: commonidentity.Identity{ID: testAgentID}})
	customerEvent := event(commonoutline.ServiceNameCustomerManager, cmcustomer.EventTypeCustomerDeleted, &cmcustomer.Customer{ID: testCustomerID})

	tests := []struct {
		name   string
		event  *sock.Event
		expect func(m *testMocks)
	}{
		{
			"agent_deleted deletes the devices and the settings of the agent",
			agentEvent,
			func(m *testMocks) {
				m.device.EXPECT().EventAgentDeleted(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *amagent.Agent) error {
					if a.ID != testAgentID {
						t.Errorf("Wrong match. expect: %s, got: %s", testAgentID, a.ID)
					}
					return nil
				})
				m.settings.EXPECT().EventAgentDeleted(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *amagent.Agent) error {
					if a.ID != testAgentID {
						t.Errorf("Wrong match. expect: %s, got: %s", testAgentID, a.ID)
					}
					return nil
				})
			},
		},
		{
			"customer_deleted deletes the devices and the settings of the customer",
			customerEvent,
			func(m *testMocks) {
				m.device.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, c *cmcustomer.Customer) error {
					if c.ID != testCustomerID {
						t.Errorf("Wrong match. expect: %s, got: %s", testCustomerID, c.ID)
					}
					return nil
				})
				m.settings.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			"the failure of the devices does not skip the settings: the error is logged, not raised",
			agentEvent,
			func(m *testMocks) {
				m.device.EXPECT().EventAgentDeleted(gomock.Any(), gomock.Any()).Return(fmt.Errorf("boom"))
				m.settings.EXPECT().EventAgentDeleted(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			"the failure of the settings does not skip the devices",
			customerEvent,
			func(m *testMocks) {
				m.device.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
				m.settings.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(fmt.Errorf("boom"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc, 4)
			tt.expect(m)

			if err := h.processEventRun(tt.event); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			waitIdle(t, h)
		})
	}
}

func Test_processEvent_events(t *testing.T) {
	tests := []struct {
		name   string
		event  *sock.Event
		expect func(m *testMocks)
	}{
		{
			"chatmessage_created",
			event(commonoutline.ServiceNameTalkManager, tkmessage.EventTypeMessageCreated, &tkmessage.Message{
				Identity: commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID},
				ChatID:   testChatID,
				Type:     tkmessage.TypeNormal,
				Text:     "hello",
			}),
			func(m *testMocks) {
				m.event.EXPECT().EventChatMessageCreated(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg *tkmessage.Message) error {
					if msg.ID != testMessageID || msg.CustomerID != testCustomerID || msg.ChatID != testChatID || msg.Text != "hello" {
						t.Errorf("Wrong match. got: %v", msg)
					}
					return nil
				})
			},
		},
		{
			"conversation_message_created",
			event(commonoutline.ServiceNameConversationManager, cvmessage.EventTypeMessageCreated, &cvmessage.Message{
				Identity:       commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID},
				ConversationID: testConvID,
				Direction:      cvmessage.DirectionIncoming,
			}),
			func(m *testMocks) {
				m.event.EXPECT().EventConversationMessageCreated(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, msg *cvmessage.Message) error {
					if msg.ID != testMessageID || msg.ConversationID != testConvID || msg.Direction != cvmessage.DirectionIncoming {
						t.Errorf("Wrong match. got: %v", msg)
					}
					return nil
				})
			},
		},
		{
			"case_note_created",
			event(commonoutline.ServiceNameContactManager, cmcasenote.EventTypeCaseNoteCreated, &cmcasenote.CaseNote{
				ID: testNoteID, CustomerID: testCustomerID, CaseID: testCaseID, AuthorType: cmcasenote.AuthorTypeAgent, AuthorID: &testAgentID,
			}),
			func(m *testMocks) {
				m.event.EXPECT().EventCaseNoteCreated(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, n *cmcasenote.CaseNote) error {
					if n.ID != testNoteID || n.CaseID != testCaseID || n.AuthorID == nil || *n.AuthorID != testAgentID {
						t.Errorf("Wrong match. got: %v", n)
					}
					return nil
				})
			},
		},
		{
			"case_assigned",
			event(commonoutline.ServiceNameContactManager, cmkase.EventTypeCaseAssigned, &cmkase.CaseAssignedEvent{CaseID: testCaseID, CustomerID: testCustomerID, OwnerID: testAgentID}),
			func(m *testMocks) {
				m.event.EXPECT().EventCaseAssigned(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, e *cmkase.CaseAssignedEvent) error {
					if e.CaseID != testCaseID || e.CustomerID != testCustomerID || e.OwnerID != testAgentID {
						t.Errorf("Wrong match. got: %v", e)
					}
					return nil
				})
			},
		},
		{
			"an error of the event handler is logged, not raised",
			event(commonoutline.ServiceNameContactManager, cmkase.EventTypeCaseAssigned, &cmkase.CaseAssignedEvent{CaseID: testCaseID}),
			func(m *testMocks) {
				m.event.EXPECT().EventCaseAssigned(gomock.Any(), gomock.Any()).Return(fmt.Errorf("boom"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newTestHandler(mc, 4)
			tt.expect(m)

			// the callback returns nil whatever the handler returns: an error would make the broker redeliver
			// the event and the notification would be sent twice
			if err := h.processEventRun(tt.event); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			waitIdle(t, h)
		})
	}
}

func Test_processEvent_ignored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no handler call is expected for another publisher, another type, or a malformed body
	h, _ := newTestHandler(mc, 1)

	h.processEvent(&sock.Event{Publisher: "call-manager", Type: amagent.EventTypeAgentDeleted})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: "agent_created"})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameAgentManager), Type: amagent.EventTypeAgentDeleted, Data: []byte("not json")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: cmcustomer.EventTypeCustomerDeleted, Data: []byte("not json")})

	// the event types of the new subscriptions are bound to their publisher
	h.processEvent(&sock.Event{Publisher: "call-manager", Type: tkmessage.EventTypeMessageCreated, Data: []byte("{}")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameTalkManager), Type: cvmessage.EventTypeMessageCreated, Data: []byte("{}")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameTalkManager), Type: tkmessage.EventTypeMessageCreated, Data: []byte("not json")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameConversationManager), Type: cvmessage.EventTypeMessageCreated, Data: []byte("not json")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameContactManager), Type: cmcasenote.EventTypeCaseNoteCreated, Data: []byte("not json")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameContactManager), Type: cmkase.EventTypeCaseAssigned, Data: []byte("not json")})
	h.processEvent(&sock.Event{Publisher: string(commonoutline.ServiceNameContactManager), Type: "case_tag_added", Data: []byte("{}")})
}

// The pool holds the events in the consumer callback: when it is full, the next event waits and the broker is
// not asked for more, which is the back pressure. The slot is given back when the event is over, also when the
// handler panics.
func Test_processEventRun_pool(t *testing.T) {
	ev := event(commonoutline.ServiceNameContactManager, cmkase.EventTypeCaseAssigned, &cmkase.CaseAssignedEvent{CaseID: testCaseID})

	t.Run("a full pool makes the next event wait", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc, 1)

		started := make(chan struct{})
		release := make(chan struct{})
		var calls int
		var mu sync.Mutex
		m.event.EXPECT().EventCaseAssigned(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *cmkase.CaseAssignedEvent) error {
			mu.Lock()
			calls++
			n := calls
			mu.Unlock()
			if n == 1 {
				close(started)
				<-release
			}
			return nil
		}).Times(2)

		if err := h.processEventRun(ev); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		<-started

		returned := make(chan struct{})
		go func() {
			_ = h.processEventRun(ev)
			close(returned)
		}()

		select {
		case <-returned:
			t.Fatalf("Wrong match. the second event must wait for a free slot")
		case <-time.After(50 * time.Millisecond):
		}

		close(release)
		select {
		case <-returned:
		case <-time.After(2 * time.Second):
			t.Fatalf("Wrong match. the second event must start when the slot is free")
		}
		waitIdle(t, h)
	})

	t.Run("a panic of the handler gives the slot back", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newTestHandler(mc, 1)

		first := true
		m.event.EXPECT().EventCaseAssigned(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, *cmkase.CaseAssignedEvent) error {
			if first {
				first = false
				panic("boom")
			}
			return nil
		}).Times(2)

		_ = h.processEventRun(ev)
		_ = h.processEventRun(ev) // blocks until the panic of the first has released the slot
		waitIdle(t, h)
	})
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

			h := NewSubscribeHandler(mockSock, "q", devicehandler.NewMockDeviceHandler(mc), settingshandler.NewMockSettingsHandler(mc), eventhandler.NewMockEventHandler(mc))
			if err := h.Run(); (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			// let the consumer goroutine reach its mock call
			time.Sleep(20 * time.Millisecond)
		})
	}
}

func Test_NewSubscribeHandler_pool(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := NewSubscribeHandler(sockhandler.NewMockSockHandler(mc), "q", devicehandler.NewMockDeviceHandler(mc), settingshandler.NewMockSettingsHandler(mc), eventhandler.NewMockEventHandler(mc)).(*subscribeHandler)
	if cap(h.sem) != maxConcurrentEvents || maxConcurrentEvents != 32 {
		t.Errorf("Wrong match. expect: 32 slots, got: %d", cap(h.sem))
	}
}
