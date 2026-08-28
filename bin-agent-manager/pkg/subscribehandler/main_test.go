package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_BindsTopicExchangeBeforeReturning is a regression test for a production incident
// (VOIP-1258 PR #1101, found during post-deploy verification 2026-07-14): QueueBind/QueueUnbind
// for the topic-exchange cutovers MUST complete synchronously inside Run(), before the async
// ConsumeMessage goroutine is started. QueueBind/QueueUnbind and ConsumeMessage's internal
// channel.Consume() share the same underlying AMQP channel for a given queue name; if a caller
// invoked QueueBind/QueueUnbind AFTER Run() returns (as this code originally did, from
// cmd/agent-manager/main.go), it races the already-started basic.consume RPC on the same
// channel and the broker can close the channel with "unexpected command received" (503) --
// silently preventing that pod from ever consuming events.
//
// Since VOIP-1406 the strict InOrder chain covers the FULL sequence:
// QueueCreate -> fanout QueueSubscribes -> [VOIP-1258 webhook-topic bind/unbind] ->
// TopicCreateWithKind(bin-manager.event) -> QueueBind(every topicPatterns entry) ->
// QueueUnbind(every fanoutUnbindTargets entry) -- and the channel-based harness asserts
// all of it completes before ConsumeMessage is observed at all.
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
	callOrderCh := make(chan string, 16)

	// number of synchronous broker operations that must ALL be recorded before
	// ConsumeMessage may fire: QueueCreate + 2 fanout QueueSubscribe + 1258
	// QueueBind/QueueUnbind + TopicCreateWithKind + len(topicPatterns) QueueBinds +
	// len(fanoutUnbindTargets) QueueUnbinds.
	syncOps := 1 + len(subscribeTargets) + 2 + 1 + len(topicPatterns) + len(fanoutUnbindTargets)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "QueueCreate"
				return nil
			}),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "QueueSubscribe"
				return nil
			}))
	}

	// the existing VOIP-1258 webhook-topic cutover block, retained untouched.
	calls = append(calls,
		mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).
			DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
				callOrderCh <- "QueueBind:1258"
				return nil
			}),
		mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).
			DoAndReturn(func(_, _, _ string, _ interface{}) error {
				callOrderCh <- "QueueUnbind:1258"
				return nil
			}),
	)

	// the VOIP-1406 bin-manager.event block: declare, bind every pattern, unbind
	// every fanout target -- strictly after the 1258 block, before ConsumeMessage.
	calls = append(calls,
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "TopicCreateWithKind"
				return nil
			}),
	)
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).
			DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
				callOrderCh <- "QueueBind:topic"
				return nil
			}))
	}
	for _, target := range fanoutUnbindTargets {
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).
			DoAndReturn(func(_, _, _ string, _ interface{}) error {
				callOrderCh <- "QueueUnbind:fanout"
				return nil
			}))
	}
	gomock.InOrder(calls...)

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

	// By the time Run() returns, every synchronous broker operation must already have been
	// recorded (they are called before the ConsumeMessage goroutine is launched). Drain
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

	seenSync := 0
	for _, c := range callOrder {
		// If ConsumeMessage's goroutine happened to race ahead and got recorded before
		// the full synchronous chain completed, the ordering bug is back.
		if c == "ConsumeMessage" {
			if seenSync < syncOps {
				t.Fatalf("ConsumeMessage was observed before the synchronous bind/unbind chain completed (%d/%d) -- ordering regression. callOrder: %v", seenSync, syncOps, callOrder)
			}
			continue
		}
		seenSync++
	}
	if seenSync != syncOps {
		t.Errorf("Expected %d synchronous broker operations to have been recorded within Run(), got %d. callOrder: %v", syncOps, seenSync, callOrder)
	}
}

// Test_Run_QueueBindFailure_DoesNotUnbind verifies the safe-failure path of the retained
// VOIP-1258 block: if QueueBind to the webhook topic exchange fails, Run() must NOT proceed
// to QueueUnbind the old webhook fanout exchange -- staying bound to the old exchange is
// safer than ending up bound to neither. The independent VOIP-1406 bin-manager.event block
// still runs to completion afterwards (the two cutovers must not affect each other).
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
	// The webhook fanout QueueUnbind must NOT be called.
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Times(0)

	// The VOIP-1406 block is independent of the 1258 failure and completes normally.
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	for _, pattern := range topicPatterns {
		mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	}
	for _, target := range fanoutUnbindTargets {
		mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).Return(nil)
	}

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
