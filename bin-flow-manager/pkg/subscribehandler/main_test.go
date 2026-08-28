package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_sequencing verifies the exact broker call sequence of Run() (VOIP-1406):
//
//	QueueCreate -> fanout QueueSubscribe (all subscribeTargets) -> TopicCreateWithKind
//	-> QueueBind (every topicPatterns entry, in order) -> QueueUnbind (every
//	fanoutUnbindTargets entry, in order).
//
// The topic block MUST run synchronously inside Run(), before the ConsumeMessage
// goroutine: QueueBind/QueueUnbind and ConsumeMessage's internal basic.consume share the
// same AMQP channel, and racing them closes the channel with a 503 (production incident
// 2026-07-14, VOIP-1258). The strict gomock controller fails the test on any call outside
// the expected set; gomock.InOrder fails it on any reordering.
func Test_Run_sequencing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameFlowSubscribe)
	subscribeTargets := []string{
		string(commonoutline.QueueNameCustomerEvent),
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	for _, target := range fanoutUnbindTargets {
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run(); it may or may not have been
	// scheduled by the time Run() returns, so it stays outside the InOrder chain.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameFlowManager), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(
		mockSock,
		queueName,
		subscribeTargets,
		nil,
		nil,
	)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}
