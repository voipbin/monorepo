package subscribehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

var errTestFailure = errors.New("simulated failure")

func Test_processEventCMConfbridge_errors(t *testing.T) {
	callID := uuid.FromStringOrNil("7d5c3f66-0001-11f1-9c3a-8b4e5d6f7a01")
	cc := &conferencecall.Conferencecall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("7d5c3f66-0002-11f1-9c3a-8b4e5d6f7a02")}}

	joinedEvent := func(data string) *sock.Event {
		return &sock.Event{Type: cmconfbridge.EventTypeConfbridgeJoined, Publisher: "call-manager", DataType: "application/json", Data: []byte(data)}
	}
	leavedEvent := func(data string) *sock.Event {
		return &sock.Event{Type: cmconfbridge.EventTypeConfbridgeLeaved, Publisher: "call-manager", DataType: "application/json", Data: []byte(data)}
	}

	tests := []struct {
		name  string
		call  func(h *subscribeHandler, ctx context.Context) error
		setup func(cc *conferencecallhandler.MockConferencecallHandler)

		expectErr error
		// expectAnyErr is true when an error without a known cause is expected.
		expectAnyErr bool
	}{
		{
			name: "joined: broken data is rejected before any lookup",
			call: func(h *subscribeHandler, ctx context.Context) error {
				return h.processEventCMConfbridgeJoined(ctx, joinedEvent(`{`))
			},
			expectAnyErr: true,
		},
		{
			name: "joined: a call that is not a conferencecall is ignored",
			call: func(h *subscribeHandler, ctx context.Context) error {
				return h.processEventCMConfbridgeJoined(ctx, joinedEvent(`{"joined_call_id":"`+callID.String()+`"}`))
			},
			setup: func(m *conferencecallhandler.MockConferencecallHandler) {
				m.EXPECT().GetByReferenceID(gomock.Any(), callID).Return(nil, errTestFailure)
				// Joined must not be called.
			},
		},
		{
			name: "joined: the join failure is returned",
			call: func(h *subscribeHandler, ctx context.Context) error {
				return h.processEventCMConfbridgeJoined(ctx, joinedEvent(`{"joined_call_id":"`+callID.String()+`"}`))
			},
			setup: func(m *conferencecallhandler.MockConferencecallHandler) {
				m.EXPECT().GetByReferenceID(gomock.Any(), callID).Return(cc, nil)
				m.EXPECT().Joined(gomock.Any(), cc).Return(nil, errTestFailure)
			},
			expectErr: errTestFailure,
		},
		{
			name: "leaved: broken data is rejected before any lookup",
			call: func(h *subscribeHandler, ctx context.Context) error {
				return h.processEventCMConfbridgeLeaved(ctx, leavedEvent(`{`))
			},
			expectAnyErr: true,
		},
		{
			name: "leaved: a call that is not a conferencecall is ignored",
			call: func(h *subscribeHandler, ctx context.Context) error {
				return h.processEventCMConfbridgeLeaved(ctx, leavedEvent(`{"leaved_call_id":"`+callID.String()+`"}`))
			},
			setup: func(m *conferencecallhandler.MockConferencecallHandler) {
				m.EXPECT().GetByReferenceID(gomock.Any(), callID).Return(nil, errTestFailure)
				// Terminated must not be called.
			},
		},
		{
			name: "leaved: the terminate failure is returned",
			call: func(h *subscribeHandler, ctx context.Context) error {
				return h.processEventCMConfbridgeLeaved(ctx, leavedEvent(`{"leaved_call_id":"`+callID.String()+`"}`))
			},
			setup: func(m *conferencecallhandler.MockConferencecallHandler) {
				m.EXPECT().GetByReferenceID(gomock.Any(), callID).Return(cc, nil)
				m.EXPECT().Terminated(gomock.Any(), cc).Return(nil, errTestFailure)
			},
			expectErr: errTestFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCC := conferencecallhandler.NewMockConferencecallHandler(mc)
			h := &subscribeHandler{
				sockHandler:           sockhandler.NewMockSockHandler(mc),
				conferenceHandler:     conferencehandler.NewMockConferenceHandler(mc),
				conferencecallHandler: mockCC,
			}
			if tt.setup != nil {
				tt.setup(mockCC)
			}

			err := tt.call(h, context.Background())
			switch {
			case tt.expectErr != nil:
				if !errors.Is(err, tt.expectErr) {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
				}
			case tt.expectAnyErr:
				if err == nil {
					t.Errorf("Wrong error. expect: an error, got: nil")
				}
			default:
				if err != nil {
					t.Errorf("Wrong error. expect: nil, got: %v", err)
				}
			}
		})
	}
}

func Test_processEvent_unknownEventIsIgnored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no handler has an expectation, so any call fails the test.
	h := &subscribeHandler{
		sockHandler:           sockhandler.NewMockSockHandler(mc),
		conferenceHandler:     conferencehandler.NewMockConferenceHandler(mc),
		conferencecallHandler: conferencecallhandler.NewMockConferencecallHandler(mc),
	}

	h.processEvent(&sock.Event{Type: "other_event", Publisher: "call-manager", Data: []byte(`{}`)})
	// the same type from another publisher is not handled either.
	h.processEvent(&sock.Event{Type: cmconfbridge.EventTypeConfbridgeJoined, Publisher: "other-publisher", Data: []byte(`{}`)})
}

func Test_processEventRun_returnsImmediately(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &subscribeHandler{
		sockHandler:           sockhandler.NewMockSockHandler(mc),
		conferenceHandler:     conferencehandler.NewMockConferenceHandler(mc),
		conferencecallHandler: conferencecallhandler.NewMockConferencecallHandler(mc),
	}

	if err := h.processEventRun(&sock.Event{Type: "other_event", Publisher: "call-manager"}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)
	mockCC := conferencecallhandler.NewMockConferencecallHandler(mc)

	h, ok := NewSubscribeHandler(mockSock, "test-queue", mockConf, mockCC).(*subscribeHandler)
	if !ok {
		t.Fatalf("Wrong type")
	}
	if h.subscribeQueue != "test-queue" || h.sockHandler != mockSock || h.conferenceHandler != mockConf || h.conferencecallHandler != mockCC {
		t.Errorf("Wrong handler wiring")
	}
}

func Test_Run(t *testing.T) {
	event := string(commonoutline.QueueNameEvent)

	t.Run("queue creation fails: nothing is bound", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: "test-queue"}
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errTestFailure)

		if err := h.Run(); !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
		}
	})

	t.Run("topic exchange creation fails: nothing is bound", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: "test-queue"}
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		mockSock.EXPECT().TopicCreateWithKind(event, "topic").Return(errTestFailure)

		if err := h.Run(); !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
		}
	})

	t.Run("a bind failure stops at the failing pattern and no consumer is started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: "test-queue"}
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		mockSock.EXPECT().TopicCreateWithKind(event, "topic").Return(nil)
		mockSock.EXPECT().QueueBind("test-queue", topicPatterns[0], event, false, nil).Return(errTestFailure)

		if err := h.Run(); !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
		}
	})

	t.Run("every pattern is bound and the consumer is started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: "test-queue"}
		done := make(chan struct{})
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		mockSock.EXPECT().TopicCreateWithKind(event, "topic").Return(nil)
		for _, pattern := range topicPatterns {
			mockSock.EXPECT().QueueBind("test-queue", pattern, event, false, nil).Return(nil)
		}
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), "test-queue", "conference-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
			func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgConsume) error {
				defer close(done)
				return errTestFailure
			})

		if err := h.Run(); err != nil {
			t.Fatalf("Wrong error. expect: nil, got: %v", err)
		}
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out waiting for ConsumeMessage to be called")
		}
	})
}
