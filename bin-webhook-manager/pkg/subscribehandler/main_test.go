package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

// Test_Run_TopicMigrationSequencing asserts the full VOIP-1407 broker call sequence inside
// Run(), in strict order: QueueCreate -> TopicCreateWithKind -> each topic QueueBind (all
// patterns). The whole sequence MUST complete synchronously before the ConsumeMessage
// goroutine starts: QueueBind and basic.consume share one AMQP channel, and racing them
// closes the channel with a 503 (VOIP-1258 production incident, 2026-07-14).
func Test_Run_TopicMigrationSequencing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	queueName := string(commonoutline.QueueNameWebhookSubscribe)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run() -- it may or may not have been
	// scheduled by the time Run() returns, so expect it loosely outside the ordered chain
	// (matching the bin-agent-manager Run() test idiom).
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "webhook-manager", false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, mockAccount, mockCache)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}
