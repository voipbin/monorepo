package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_BindsTopicExchangeBeforeReturning is a regression test for a production incident
// (VOIP-1258 PR #1101, found during post-deploy verification 2026-07-14): QueueBind/QueueUnbind
// for the topic-exchange cutover MUST complete synchronously inside Run(), before the async
// ConsumeMessage goroutine is started. QueueBind/QueueUnbind and ConsumeMessage's internal
// channel.Consume() share the same underlying AMQP channel for a given queue name; if a caller
// invoked QueueBind/QueueUnbind AFTER Run() returns (as this code originally did, from
// cmd/agent-manager/main.go), it races the already-started basic.consume RPC on the same
// channel and the broker can close the channel with "unexpected command received" (503) --
// silently preventing that pod from ever consuming events. This test asserts the ordering:
// QueueBind/QueueUnbind must both be called before ConsumeMessage is invoked at all.
func Test_Run_BindsTopicExchangeBeforeReturning(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameAgentSubscribe)
	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}

	// callOrderCh, not a shared slice: QueueBind/QueueUnbind run synchronously on the caller's
	// goroutine, but ConsumeMessage's callback runs on Run()'s internal goroutine. A plain
	// `var callOrder []string` appended from both would be a data race (caught by `-race`,
	// intermittently, since ConsumeMessage's goroutine may or may not have scheduled by the
	// time Run() returns) -- exactly the bug class this production fix addresses, ironically
	// reintroduced in the test harness in an earlier version of this test. Use a buffered
	// channel instead, matching bin-timeline-manager's Test_Run_BindsTopicExchangeBeforeConsuming.
	callOrderCh := make(chan string, 8)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	for _, target := range subscribeTargets {
		mockSock.EXPECT().QueueSubscribe(queueName, target).Return(nil)
	}
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).
		DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
			callOrderCh <- "QueueBind"
			return nil
		})
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).
		DoAndReturn(func(_, _, _ string, _ interface{}) error {
			callOrderCh <- "QueueUnbind"
			return nil
		})
	// ConsumeMessage is started in a goroutine inside Run() -- allow it to be called (or not,
	// if the goroutine hasn't scheduled yet by the time Run() returns) without blocking the
	// test; the ordering assertion below is what actually matters, not whether this fires.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			callOrderCh <- "ConsumeMessage"
			return nil
		}).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	// By the time Run() returns, QueueBind and QueueUnbind must already have been recorded
	// (they are called synchronously before the ConsumeMessage goroutine is launched). Drain
	// whatever has arrived so far without closing the channel -- ConsumeMessage's mock may
	// still be in flight on its own goroutine and could send after we're done reading.
	var callOrder []string
	draining := true
	for draining {
		select {
		case c := <-callOrderCh:
			callOrder = append(callOrder, c)
		default:
			draining = false
		}
	}
	foundBind, foundUnbind := false, false
	for _, c := range callOrder {
		if c == "QueueBind" {
			foundBind = true
		}
		if c == "QueueUnbind" {
			foundUnbind = true
		}
		// If ConsumeMessage's goroutine happened to race ahead and got recorded before
		// QueueBind/QueueUnbind, that would indicate the ordering bug is back.
		if c == "ConsumeMessage" && (!foundBind || !foundUnbind) {
			t.Fatalf("ConsumeMessage was observed before QueueBind/QueueUnbind completed -- ordering regression. callOrder: %v", callOrder)
		}
	}
	if !foundBind {
		t.Errorf("Expected QueueBind to have been called synchronously within Run(), callOrder: %v", callOrder)
	}
	if !foundUnbind {
		t.Errorf("Expected QueueUnbind to have been called synchronously within Run(), callOrder: %v", callOrder)
	}
}

// Test_Run_QueueBindFailure_DoesNotUnbind verifies the safe-failure path: if QueueBind to the
// new topic exchange fails, Run() must NOT proceed to QueueUnbind the old fanout exchange --
// staying bound to the old exchange is safer than ending up bound to neither.
func Test_Run_QueueBindFailure_DoesNotUnbind(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameAgentSubscribe)
	subscribeTargets := []string{string(commonoutline.QueueNameCallEvent)}

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, subscribeTargets[0]).Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).
		Return(assertError("bind failed"))
	// QueueUnbind must NOT be called.
	mockSock.EXPECT().QueueUnbind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
