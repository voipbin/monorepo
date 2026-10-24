package subscribehandler

import (
	"errors"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"
)

func Test_Run_errors(t *testing.T) {
	queueName := "test-subscribe-queue"
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name   string
		expect func(m *sockhandler.MockSockHandler)
	}{
		{
			name: "queue create fails",
			expect: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queueName, "normal").Return(errors.New("queue failed"))
			},
		},
		{
			name: "topic create fails",
			expect: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(errors.New("topic failed")),
				)
			},
		},
		{
			// the failure is on the second pattern: the first is bound, the third is never
			// attempted (it has no expectation) and the consumer is never started.
			name: "queue bind fails on the second pattern",
			expect: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queueName, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[0], exchange, false, nil).Return(nil),
					m.EXPECT().QueueBind(queueName, topicPatterns[1], exchange, false, nil).Return(errors.New("bind failed")),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.expect(mockSock)

			h := NewSubscribeHandler("transfer-manager", mockSock, queueName, nil)
			if err := h.Run(); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_Run_consumeErrorIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := logrustest.NewGlobal()
	defer hook.Reset()

	mockSock := sockhandler.NewMockSockHandler(mc)
	queueName := "test-subscribe-queue"
	exchange := string(commonoutline.QueueNameEvent)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil)
	for _, pattern := range topicPatterns {
		mockSock.EXPECT().QueueBind(queueName, pattern, exchange, false, nil).Return(nil)
	}
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "transfer-manager", false, false, false, 10, gomock.Any()).Return(errors.New("consume failed"))

	h := NewSubscribeHandler("transfer-manager", mockSock, queueName, nil)
	if err := h.Run(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	waitForLog(t, hook, "Could not consume the request message correctly")
}

func Test_processEvent_handlerErrorIsLogged(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := logrustest.NewGlobal()
	defer hook.Reset()

	// no expectation on the transfer handler: an unmarshal failure stops before any lookup.
	h, _ := newErrorTestHandler(mc)

	h.processEvent(&sock.Event{
		Publisher: publisherCallManager,
		Type:      "groupcall_progressing",
		Data:      []byte(`{invalid json`),
	})

	waitForLog(t, hook, "Could not process the event correctly")
}

// waitForLog waits until a log entry containing the message is emitted.
func waitForLog(t *testing.T, hook *logrustest.Hook, msg string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.ErrorLevel && strings.Contains(e.Message, msg) {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("Wrong match. expect: error log containing %q, got: none", msg)
}
