package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-api-manager/pkg/zmqpubhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_BindsTopicExchangeBeforeReturning is a regression test for a production incident
// (VOIP-1258 PR #1101, bin-agent-manager, found during post-deploy verification 2026-07-14,
// fixed in commit ca8c104a9 and proactively applied here too): QueueBind for the topic-exchange
// baseline "#" wildcard MUST complete synchronously inside Run(), before the async
// ConsumeMessage goroutine is started. QueueBind and ConsumeMessage's internal
// channel.Consume() share the same underlying AMQP channel for a given queue name; if a caller
// invoked QueueBind AFTER Run() returns (as this code originally did, from
// cmd/api-manager/main.go), it races the already-started basic.consume RPC on the same channel
// and the broker can close the channel with "unexpected command received" (503) -- silently
// preventing that pod from ever consuming events. This test asserts the ordering: QueueBind
// must be called before ConsumeMessage is observed.
func Test_Run_BindsTopicExchangeBeforeReturning(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockZmq := zmqpubhandler.NewMockZMQPubHandler(mc)

	queueName := "bin-manager.api-manager.subscribe-test-pod"
	subscribeTargets := []string{}

	var callOrder []string

	mockSock.EXPECT().QueueCreate(queueName, "volatile").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).
		DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
			callOrder = append(callOrder, "QueueBind")
			return nil
		})
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			callOrder = append(callOrder, "ConsumeMessage")
			return nil
		}).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockReq, queueName, subscribeTargets, mockZmq)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	foundBind := false
	for _, c := range callOrder {
		if c == "QueueBind" {
			foundBind = true
		}
		if c == "ConsumeMessage" && !foundBind {
			t.Fatalf("ConsumeMessage was observed before QueueBind completed -- ordering regression. callOrder: %v", callOrder)
		}
	}
	if !foundBind {
		t.Errorf("Expected QueueBind to have been called synchronously within Run(), callOrder: %v", callOrder)
	}
}
