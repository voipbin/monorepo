package subscribehandler

import (
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// The tests below pin the Run() sequence of VOIP-1407: queue create -> topic exchange
// declare -> pattern binds -> consume. The bind block MUST complete synchronously
// inside Run(), before the async ConsumeMessage goroutine starts: QueueBind and
// ConsumeMessage's internal channel.Consume() share the same underlying AMQP channel
// for a given queue, and racing them makes the broker close the channel with a 503
// (VOIP-1258 PR #1101 post-deploy incident, 2026-07-14). There is no fanout fallback
// left to degrade to, so every startup failure returns immediately.

// Test_Run_sequencing_success verifies the full happy path in strict order:
// QueueCreate -> TopicCreateWithKind -> QueueBind for every topic pattern.
func Test_Run_sequencing_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run(); it may or may not have
	// been scheduled by the time Run() returns.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameContactManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

// Test_Run_sequencing_declareFailure verifies that a topic exchange declare failure
// returns the error immediately: zero topic binds happen, and Run() fails.
func Test_Run_sequencing_declareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("declare failed")),
	}
	gomock.InOrder(calls...)

	mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	h := NewSubscribeHandler(mockSock, queueName, nil)

	if err := h.Run(); err == nil {
		t.Fatalf("Run() expected an error, got nil")
	}
}

// Test_Run_sequencing_bindFailure_returnsErrorImmediately verifies that a topic
// pattern bind failure returns the error immediately: no rollback exists any more
// (VOIP-1407 removed the "bound so far, roll back on partial failure" machinery),
// and no fanout exchange exists to fall back to.
func Test_Run_sequencing_bindFailure_returnsErrorImmediately(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameContactSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
		mockSock.EXPECT().QueueBind(queueName, topicPatterns[0], string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("bind failed")),
	}
	gomock.InOrder(calls...)

	// no rollback: QueueUnbind is never called.
	mockSock.EXPECT().QueueUnbind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	h := NewSubscribeHandler(mockSock, queueName, nil)

	if err := h.Run(); err == nil {
		t.Fatalf("Run() expected an error, got nil")
	}
}
