package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

func TestNewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	tests := []struct {
		name           string
		serviceName    string
		subscribeQueue string
	}{
		{
			name:           "creates_handler_successfully",
			serviceName:    "transfer-manager",
			subscribeQueue: "test-queue",
		},
		{
			name:           "creates_handler_with_empty_targets",
			serviceName:    "transfer-manager",
			subscribeQueue: "test-queue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewSubscribeHandler(
				tt.serviceName,
				mockSock,
				tt.subscribeQueue,
				mockTransfer,
			)

			if h == nil {
				t.Error("Expected handler but got nil")
			}
		})
	}
}

func TestProcessEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTransfer := transferhandler.NewMockTransferHandler(mc)

	h := &subscribeHandler{
		serviceName:     "transfer-manager",
		sockHandler:     mockSock,
		subscribeQueue:  "test-queue",
		transferHandler: mockTransfer,
	}

	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			name: "processes_event_without_error",
			event: &sock.Event{
				Publisher: "unknown-publisher",
				Type:      "unknown-type",
				Data:      []byte(`{}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.processEventRun(tt.event)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// Test_Run_sequencing verifies the exact broker call sequence of Run() (VOIP-1407):
//
//	QueueCreate -> TopicCreateWithKind -> QueueBind (every topicPatterns entry, in order).
//
// The topic block MUST run synchronously inside Run(), before the ConsumeMessage
// goroutine: QueueBind and ConsumeMessage's internal basic.consume share the same AMQP
// channel, and racing them closes the channel with a 503 (production incident
// 2026-07-14, VOIP-1258). The strict gomock controller fails the test on any call outside
// the expected set; gomock.InOrder fails it on any reordering.
func Test_Run_sequencing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameTransferSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run(); it may or may not have been
	// scheduled by the time Run() returns, so it stays outside the InOrder chain.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameTransferManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(
		string(commonoutline.ServiceNameTransferManager),
		mockSock,
		queueName,
		nil,
	)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}
