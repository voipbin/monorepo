package subscribehandler

import (
	"context"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/pkg/dbhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_BindsTopicExchangeBeforeConsuming is a regression test guarding against the same
// production incident found in bin-agent-manager (VOIP-1258 PR #1101, 2026-07-14, fixed in
// commit ca8c104a9): the topic-exchange cutover's QueueBind/QueueUnbind and ConsumeMessage's
// internal channel.Consume() share the same underlying AMQP channel for a given queue name.
// If QueueBind/QueueUnbind were ever moved to run concurrently with (or after) the async
// ConsumeMessage goroutine, the broker could close the channel with "unexpected command
// received" (503), silently preventing this pod from ever consuming events.
//
// This service's Run() currently sequences QueueBind/QueueUnbind synchronously BEFORE starting
// the ConsumeMessage goroutine, which is safe. This test locks that ordering in so a future
// refactor cannot silently reintroduce the race.
func Test_Run_BindsTopicExchangeBeforeConsuming(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	var callOrder []string
	callOrderCh := make(chan string, len(subscribeTargets)+10)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	for _, target := range subscribeTargets {
		mockSock.EXPECT().QueueSubscribe(queueName, string(target)).Return(nil)
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
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			callOrderCh <- "ConsumeMessage"
			return nil
		}).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	// By the time Run() returns, QueueBind and QueueUnbind must already have been recorded --
	// they run synchronously inside Run(), before the ConsumeMessage goroutine is launched.
	// Drain whatever has arrived so far without closing the channel (ConsumeMessage's mock may
	// still be in flight on its own goroutine and could send after we're done reading).
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

	cancel()
	<-doneCh
}
