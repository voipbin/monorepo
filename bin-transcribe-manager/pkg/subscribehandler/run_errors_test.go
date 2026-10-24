package subscribehandler

import (
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	cmcall "monorepo/bin-call-manager/models/call"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
)

func Test_Run_declareFailures(t *testing.T) {
	queueName := string(commonoutline.QueueNameTranscribeSubscribe)
	eventExchange := string(commonoutline.QueueNameEvent)
	firstPattern := topicPatterns[0]

	tests := []struct {
		name string

		setup func(mockSock *sockhandler.MockSockHandler)
	}{
		{
			name: "queue declaration failure stops before any topic work",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				mockSock.EXPECT().QueueCreate(queueName, "normal").Return(fmt.Errorf("declare failed"))
			},
		},
		{
			name: "topic exchange declaration failure stops before any binding",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					mockSock.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(fmt.Errorf("declare failed")),
				)
			},
		},
		{
			name: "binding failure stops at the first failing pattern",
			setup: func(mockSock *sockhandler.MockSockHandler) {
				gomock.InOrder(
					mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					mockSock.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(nil),
					mockSock.EXPECT().QueueBind(queueName, firstPattern, eventExchange, false, nil).Return(fmt.Errorf("bind failed")),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.setup(mockSock)

			// no ConsumeMessage expectation: the consumer must not start after a failure
			h := NewSubscribeHandler(mockSock, commonoutline.QueueNameTranscribeSubscribe, nil)
			if err := h.Run(); err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_processEventRun_swallowsErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the transcribe handler has no expectation: the payload is rejected before it is reached
	h := &subscribeHandler{transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc)}

	event := &sock.Event{
		Publisher: string(commonoutline.ServiceNameCallManager),
		Type:      cmcall.EventTypeCallHangup,
		Data:      []byte(`{not-json`),
	}

	// processEvent reports the unmarshal failure, but the consumer callback must acknowledge the message
	if err := h.processEvent(event); err == nil {
		t.Errorf("Wrong match. processEvent expect: err, got: ok")
	}
	if err := h.processEventRun(event); err != nil {
		t.Errorf("Wrong match. processEventRun expect: ok (error swallowed), got: %v", err)
	}
}

func Test_processEvent_ignoresUnknownEvents(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &subscribeHandler{transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc)}

	tests := []*sock.Event{
		{Publisher: string(commonoutline.ServiceNameCallManager), Type: "call_unknown_event", Data: []byte(`{}`)},
		{Publisher: "unknown-manager", Type: cmcall.EventTypeCallHangup, Data: []byte(`{}`)},
	}

	for _, event := range tests {
		if err := h.processEvent(event); err != nil {
			t.Errorf("Wrong match. publisher: %s, type: %s, expect: ok, got: %v", event.Publisher, event.Type, err)
		}
	}
}
