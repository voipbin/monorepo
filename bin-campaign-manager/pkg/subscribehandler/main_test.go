package subscribehandler

import (
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_BindsTopicExchangeBeforeConsuming verifies the VOIP-1407 sequence inside
// Run() with a strict InOrder chain:
//
//	QueueCreate -> TopicCreateWithKind(bin-manager.event) -> QueueBind(every
//	topicPatterns entry)
//
// and asserts all of it completes synchronously before ConsumeMessage is observed
// at all. The bind RPCs and ConsumeMessage's internal basic.consume share the
// queue's AMQP channel; running them after the consume goroutine starts can close
// the channel with a 503 (the VOIP-1258 2026-07-14 production race), so the
// ordering is load-bearing, not cosmetic.
func Test_Run_BindsTopicExchangeBeforeConsuming(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	queueName := string(commonoutline.QueueNameCampaignSubscribe)

	// callOrderCh, not a shared slice: the broker RPCs run synchronously on the
	// caller's goroutine, but ConsumeMessage's callback runs on Run()'s internal
	// goroutine -- a plain slice appended from both would be a data race. Matches
	// bin-agent-manager's Test_Run_BindsTopicExchangeBeforeReturning harness.
	callOrderCh := make(chan string, 16)

	// number of synchronous broker operations that must ALL be recorded before
	// ConsumeMessage may fire: QueueCreate + TopicCreateWithKind +
	// len(topicPatterns) QueueBinds.
	syncOps := 1 + 1 + len(topicPatterns)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "QueueCreate"
				return nil
			}),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "TopicCreateWithKind"
				return nil
			}),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).
			DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
				callOrderCh <- "QueueBind:topic"
				return nil
			}))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run() -- allow it to be
	// called (or not, if the goroutine hasn't scheduled yet by the time Run()
	// returns) without blocking the test; the ordering assertion below is what
	// actually matters, not whether this fires.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameCampaignManager), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			callOrderCh <- "ConsumeMessage"
			return nil
		}).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, nil, nil, nil)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	// By the time Run() returns, every synchronous broker operation must already
	// have been recorded (they are called before the ConsumeMessage goroutine is
	// launched). Drain whatever has arrived so far without closing the channel --
	// ConsumeMessage's mock may still be in flight on its own goroutine and could
	// send after we're done reading.
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
		// If ConsumeMessage's goroutine happened to race ahead and got recorded
		// before the full synchronous chain completed, the ordering bug is back.
		if c == "ConsumeMessage" {
			if seenSync < syncOps {
				t.Fatalf("ConsumeMessage was observed before the synchronous bind chain completed (%d/%d) -- ordering regression. callOrder: %v", seenSync, syncOps, callOrder)
			}
			continue
		}
		seenSync++
	}
	if seenSync != syncOps {
		t.Errorf("Expected %d synchronous broker operations to have been recorded within Run(), got %d. callOrder: %v", syncOps, seenSync, callOrder)
	}
}
