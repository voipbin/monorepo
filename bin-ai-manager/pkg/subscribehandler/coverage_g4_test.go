package subscribehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmdtmf "monorepo/bin-call-manager/models/dtmf"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	pmmessage "monorepo/bin-pipecat-manager/models/message"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
)

const g4EvtID = "db993672-b873-11f0-bccf-37a302bcc930"

// g4Wait waits for ch to be closed or fails after a bounded timeout.
func g4Wait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatalf("timed out waiting for the detached handler")
	}
}

func g4Event(publisher, evtType, data string) *sock.Event {
	return &sock.Event{
		Publisher: publisher,
		Type:      evtType,
		DataType:  "application/json",
		Data:      []byte(data),
	}
}

func g4NewHandler(mc *gomock.Controller) (*subscribeHandler, *aicallhandler.MockAIcallHandler, *summaryhandler.MockSummaryHandler, *messagehandler.MockMessageHandler) {
	a := aicallhandler.NewMockAIcallHandler(mc)
	s := summaryhandler.NewMockSummaryHandler(mc)
	m := messagehandler.NewMockMessageHandler(mc)
	return &subscribeHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		aicallHandler:  a,
		summaryHandler: s,
		messageHandler: m,
	}, a, s, m
}

func Test_processEventCMConfbridgeJoined(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, a, _, _ := g4NewHandler(mc)

	done := make(chan struct{})
	joined := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	a.EXPECT().EventCMConfbridgeJoined(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, evt *cmconfbridge.EventConfbridgeJoined) {
			defer close(done)
			if evt.JoinedCallID != joined || evt.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("unexpected event: %+v", evt)
			}
		})

	err := h.processEventCMConfbridgeJoined(context.Background(), g4Event("call-manager", cmconfbridge.EventTypeConfbridgeJoined,
		`{"id":"`+g4EvtID+`","joined_call_id":"`+joined.String()+`"}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	g4Wait(t, done)
}

func Test_processEventCMConfbridgeLeaved(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, a, _, _ := g4NewHandler(mc)

	done := make(chan struct{})
	leaved := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	a.EXPECT().EventCMConfbridgeLeaved(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, evt *cmconfbridge.EventConfbridgeLeaved) {
			defer close(done)
			if evt.LeavedCallID != leaved || evt.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("unexpected event: %+v", evt)
			}
		})

	err := h.processEventCMConfbridgeLeaved(context.Background(), g4Event("call-manager", cmconfbridge.EventTypeConfbridgeLeaved,
		`{"id":"`+g4EvtID+`","leaved_call_id":"`+leaved.String()+`"}`))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	g4Wait(t, done)
}

func Test_processEventCMCallHangup_DispatchesBothHandlers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, a, s, _ := g4NewHandler(mc)

	doneAI := make(chan struct{})
	doneSummary := make(chan struct{})
	a.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *cmcall.Call) {
			defer close(doneAI)
			if c.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("aicall handler got unexpected call: %+v", c)
			}
		})
	s.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, c *cmcall.Call) {
			defer close(doneSummary)
			if c.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("summary handler got unexpected call: %+v", c)
			}
		})

	if err := h.processEventCMCallHangup(context.Background(), g4Event("call-manager", string(cmcall.EventTypeCallHangup), `{"id":"`+g4EvtID+`"}`)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	g4Wait(t, doneAI)
	g4Wait(t, doneSummary)
}

func Test_processEventCM_InvalidJSON_NoHandlerCalled(t *testing.T) {
	tests := []struct {
		name string
		call func(h *subscribeHandler, m *sock.Event) error
	}{
		{"confbridge joined", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMConfbridgeJoined(context.Background(), m)
		}},
		{"confbridge leaved", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMConfbridgeLeaved(context.Background(), m)
		}},
		{"call hangup", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMCallHangup(context.Background(), m)
		}},
		{"dtmf received", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCMDTMFReceived(context.Background(), m)
		}},
		{"pipecatcall initialized", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventPMPipecatcallInitialized(context.Background(), m)
		}},
		{"pipecatcall terminated", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventPMPipecatcallTerminated(context.Background(), m)
		}},
		{"conversation message created", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventCVMessageCreated(context.Background(), m)
		}},
		{"transcript created", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventTMTranscriptCreated(context.Background(), m)
		}},
		{"bot llm intermediate", func(h *subscribeHandler, m *sock.Event) error {
			return h.processEventPMMessageBotLLMIntermediate(context.Background(), m)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			// Mocks carry no expectations: any hand-off would fail the test.
			h, _, _, _ := g4NewHandler(mc)

			if err := tt.call(h, g4Event("any", "any", `{invalid`)); err == nil {
				t.Errorf("expected an unmarshal error")
			}
		})
	}
}

func Test_processEventCVMessageCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, a, _, _ := g4NewHandler(mc)

	a.EXPECT().EventCVMessageCreated(gomock.Any(), gomock.Any()).Do(
		func(_ context.Context, evt *cvmessage.Message) {
			if evt.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("unexpected event: %+v", evt)
			}
		})

	if err := h.processEventCVMessageCreated(context.Background(), g4Event("conversation-manager", cvmessage.EventTypeMessageCreated, `{"id":"`+g4EvtID+`"}`)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func Test_processEventTMTranscriptCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, a, _, _ := g4NewHandler(mc)

	a.EXPECT().EventTMTranscriptCreated(gomock.Any(), gomock.Any()).Do(
		func(_ context.Context, evt *tmtranscript.Transcript) {
			if evt.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("unexpected event: %+v", evt)
			}
		})

	if err := h.processEventTMTranscriptCreated(context.Background(), g4Event("transcribe-manager", tmtranscript.EventTypeTranscriptCreated, `{"id":"`+g4EvtID+`"}`)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func Test_processEventPMMessageBotLLMIntermediate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, _, m := g4NewHandler(mc)

	m.EXPECT().EventPMMessageBotLLMIntermediate(gomock.Any(), gomock.Any()).Do(
		func(_ context.Context, evt *pmmessage.Message) {
			if evt.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("unexpected event: %+v", evt)
			}
		})

	if err := h.processEventPMMessageBotLLMIntermediate(context.Background(), g4Event("pipecat-manager", string(pmmessage.EventTypeBotLLMIntermediate), `{"id":"`+g4EvtID+`"}`)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// Test_processEvent_Dispatch checks that each publisher/type pair reaches the intended handler.
func Test_processEvent_Dispatch(t *testing.T) {
	data := `{"id":"` + g4EvtID + `"}`
	tests := []struct {
		name  string
		event *sock.Event
		// expect registers the expected mock call and returns the channel closed once it ran.
		expect func(a *aicallhandler.MockAIcallHandler, s *summaryhandler.MockSummaryHandler, m *messagehandler.MockMessageHandler) []chan struct{}
	}{
		{
			name:  "confbridge joined",
			event: g4Event("call-manager", string(cmconfbridge.EventTypeConfbridgeJoined), data),
			expect: func(a *aicallhandler.MockAIcallHandler, _ *summaryhandler.MockSummaryHandler, _ *messagehandler.MockMessageHandler) []chan struct{} {
				d := make(chan struct{})
				a.EXPECT().EventCMConfbridgeJoined(gomock.Any(), gomock.Any()).Do(func(context.Context, *cmconfbridge.EventConfbridgeJoined) { close(d) })
				return []chan struct{}{d}
			},
		},
		{
			name:  "confbridge leaved",
			event: g4Event("call-manager", string(cmconfbridge.EventTypeConfbridgeLeaved), data),
			expect: func(a *aicallhandler.MockAIcallHandler, _ *summaryhandler.MockSummaryHandler, _ *messagehandler.MockMessageHandler) []chan struct{} {
				d := make(chan struct{})
				a.EXPECT().EventCMConfbridgeLeaved(gomock.Any(), gomock.Any()).Do(func(context.Context, *cmconfbridge.EventConfbridgeLeaved) { close(d) })
				return []chan struct{}{d}
			},
		},
		{
			name:  "call hangup",
			event: g4Event("call-manager", string(cmcall.EventTypeCallHangup), data),
			expect: func(a *aicallhandler.MockAIcallHandler, s *summaryhandler.MockSummaryHandler, _ *messagehandler.MockMessageHandler) []chan struct{} {
				d1 := make(chan struct{})
				d2 := make(chan struct{})
				a.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).Do(func(context.Context, *cmcall.Call) { close(d1) })
				s.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).Do(func(context.Context, *cmcall.Call) { close(d2) })
				return []chan struct{}{d1, d2}
			},
		},
		{
			name:  "bot llm intermediate",
			event: g4Event("pipecat-manager", string(pmmessage.EventTypeBotLLMIntermediate), data),
			expect: func(_ *aicallhandler.MockAIcallHandler, _ *summaryhandler.MockSummaryHandler, m *messagehandler.MockMessageHandler) []chan struct{} {
				m.EXPECT().EventPMMessageBotLLMIntermediate(gomock.Any(), gomock.Any())
				return nil
			},
		},
		{
			name:  "pipecatcall initialized",
			event: g4Event("pipecat-manager", string(pmpipecatcall.EventTypeInitialized), data),
			expect: func(a *aicallhandler.MockAIcallHandler, _ *summaryhandler.MockSummaryHandler, _ *messagehandler.MockMessageHandler) []chan struct{} {
				d := make(chan struct{})
				a.EXPECT().EventPMPipecatcallInitialized(gomock.Any(), gomock.Any()).Do(func(context.Context, *pmpipecatcall.Pipecatcall) { close(d) })
				return []chan struct{}{d}
			},
		},
		{
			name:  "transcript created",
			event: g4Event("transcribe-manager", tmtranscript.EventTypeTranscriptCreated, data),
			expect: func(a *aicallhandler.MockAIcallHandler, _ *summaryhandler.MockSummaryHandler, _ *messagehandler.MockMessageHandler) []chan struct{} {
				a.EXPECT().EventTMTranscriptCreated(gomock.Any(), gomock.Any())
				return nil
			},
		},
		{
			name:  "conversation message created",
			event: g4Event("conversation-manager", cvmessage.EventTypeMessageCreated, data),
			expect: func(a *aicallhandler.MockAIcallHandler, _ *summaryhandler.MockSummaryHandler, _ *messagehandler.MockMessageHandler) []chan struct{} {
				a.EXPECT().EventCVMessageCreated(gomock.Any(), gomock.Any())
				return nil
			},
		},
		{
			name:  "unknown event is ignored",
			event: g4Event("call-manager", "unknown_event", data),
			expect: func(*aicallhandler.MockAIcallHandler, *summaryhandler.MockSummaryHandler, *messagehandler.MockMessageHandler) []chan struct{} {
				return nil
			},
		},
		{
			name:  "known type from unexpected publisher is ignored",
			event: g4Event("flow-manager", string(cmcall.EventTypeCallHangup), data),
			expect: func(*aicallhandler.MockAIcallHandler, *summaryhandler.MockSummaryHandler, *messagehandler.MockMessageHandler) []chan struct{} {
				return nil
			},
		},
		{
			name:  "handler error is logged and not propagated",
			event: g4Event("transcribe-manager", tmtranscript.EventTypeTranscriptCreated, `{invalid`),
			expect: func(*aicallhandler.MockAIcallHandler, *summaryhandler.MockSummaryHandler, *messagehandler.MockMessageHandler) []chan struct{} {
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, a, s, m := g4NewHandler(mc)

			dones := tt.expect(a, s, m)
			h.processEvent(tt.event)
			for _, d := range dones {
				g4Wait(t, d)
			}
		})
	}
}

func Test_processEventRun_RunsDetached(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, a, _, _ := g4NewHandler(mc)

	done := make(chan struct{})
	a.EXPECT().EventCMDTMFReceived(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, evt *cmdtmf.DTMF) {
			defer close(done)
			if evt.ID != uuid.FromStringOrNil(g4EvtID) {
				t.Errorf("unexpected event: %+v", evt)
			}
		})

	if err := h.processEventRun(g4Event("call-manager", string(cmdtmf.EventTypeDTMFReceived), `{"id":"`+g4EvtID+`"}`)); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	g4Wait(t, done)
}

func Test_Run_Errors(t *testing.T) {
	queueName := "test-queue"
	topicName := string(commonoutline.QueueNameEvent)
	boom := errors.New("broker error")

	tests := []struct {
		name   string
		expect func(s *sockhandler.MockSockHandler)
	}{
		{
			name: "queue create fails",
			expect: func(s *sockhandler.MockSockHandler) {
				s.EXPECT().QueueCreate(queueName, "normal").Return(boom)
			},
		},
		{
			name: "topic create fails",
			expect: func(s *sockhandler.MockSockHandler) {
				s.EXPECT().QueueCreate(queueName, "normal").Return(nil)
				s.EXPECT().TopicCreateWithKind(topicName, "topic").Return(boom)
			},
		},
		{
			name: "queue bind fails on the first pattern",
			expect: func(s *sockhandler.MockSockHandler) {
				s.EXPECT().QueueCreate(queueName, "normal").Return(nil)
				s.EXPECT().TopicCreateWithKind(topicName, "topic").Return(nil)
				s.EXPECT().QueueBind(queueName, topicPatterns[0], topicName, false, nil).Return(boom)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.expect(mockSock)

			h := NewSubscribeHandler("ai-manager", mockSock, queueName, nil, nil, nil)
			err := h.Run()
			if !errors.Is(err, boom) {
				t.Errorf("expected the broker error to be wrapped, got: %v", err)
			}
		})
	}
}

func Test_Run_ConsumeError_IsLoggedOnly(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	queueName := "test-queue"

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).Times(len(topicPatterns))

	done := make(chan struct{})
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameAIManager), false, false, false, 10, gomock.Any()).DoAndReturn(
		func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgConsume) error {
			defer close(done)
			return errors.New("consume failed")
		})

	h := NewSubscribeHandler("ai-manager", mockSock, queueName, nil, nil, nil)
	if err := h.Run(); err != nil {
		t.Fatalf("Run must not fail when the consumer goroutine fails later: %v", err)
	}
	g4Wait(t, done)
}
