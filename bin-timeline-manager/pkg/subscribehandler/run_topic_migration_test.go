package subscribehandler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/pkg/dbhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_TopicMigrationSequence pins the full VOIP-1406 call sequence inside Run()
// with strict, ordered expectations:
//
//	QueueCreate -> (sentinel TopicCreate ->) each fanout QueueSubscribe ->
//	VOIP-1258 webhook-topic QueueBind + QueueUnbind ->
//	TopicCreateWithKind(bin-manager.event) -> QueueBind("#") ->
//	QueueUnbind of every fanoutUnbindTargets entry -> ConsumeMessage.
//
// ConsumeMessage runs on its own goroutine, so it is expected separately (AnyTimes)
// with an assertion that the synchronous migration block completed first.
func Test_Run_TopicMigrationSequence(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	var migrationComplete atomic.Bool

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		if target == commonoutline.QueueNameSentinelEvent {
			calls = append(calls, mockSock.EXPECT().TopicCreate(string(target)).Return(nil))
		}
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, string(target)).Return(nil))
	}
	calls = append(calls,
		mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil),
		mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
		mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(nil),
	)
	for i, target := range fanoutUnbindTargets {
		last := i == len(fanoutUnbindTargets)-1
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", string(target), nil).
			DoAndReturn(func(_, _, _ string, _ interface{}) error {
				if last {
					migrationComplete.Store(true)
				}
				return nil
			}))
	}
	gomock.InOrder(calls...)

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			if !migrationComplete.Load() {
				t.Errorf("ConsumeMessage started before the topic migration block completed -- ordering regression.")
			}
			return nil
		}).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}
	if !migrationComplete.Load() {
		t.Errorf("Expected the topic migration block to complete synchronously within Run().")
	}

	cancel()
	<-doneCh
}

// Test_Run_TopicMigration_DeclareFailure verifies that when the global topic exchange
// declare fails, Run() stays fully on the fanout subscriptions: zero QueueBind calls on
// bin-manager.event and zero fanout QueueUnbind calls (strict mock -- any such call
// fails the test), and Run() still succeeds.
func Test_Run_TopicMigration_DeclareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameSentinelEvent)).Return(nil)
	for _, target := range subscribeTargets {
		mockSock.EXPECT().QueueSubscribe(queueName, string(target)).Return(nil)
	}
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("declare failed"))
	// NO QueueBind on bin-manager.event and NO fanout QueueUnbind expectations: the
	// strict mock rejects any unexpected call, proving the whole block is skipped.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() must succeed when the topic declare fails (degrade to fanout), got err: %v", err)
	}

	cancel()
	<-doneCh
}

// Test_Run_TopicMigration_BindFailure verifies the all-or-nothing rule when a pattern
// bind fails. timeline-manager binds a single "#" pattern, so a failure of that first
// bind means bound[0..i-1] is empty: zero rollback unbinds on bin-manager.event and
// zero fanout unbinds (strict mock), and Run() still succeeds fully on fanout.
func Test_Run_TopicMigration_BindFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameSentinelEvent)).Return(nil)
	for _, target := range subscribeTargets {
		mockSock.EXPECT().QueueSubscribe(queueName, string(target)).Return(nil)
	}
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("bind failed"))
	// NO rollback QueueUnbind on bin-manager.event (nothing was bound before the
	// failure) and NO fanout QueueUnbind: the strict mock rejects any such call.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() must succeed when the pattern bind fails (degrade to fanout), got err: %v", err)
	}

	cancel()
	<-doneCh
}

// Test_Run_TopicMigration_FanoutUnbindFailure verifies that a single failed fanout
// unbind is CRITICAL-logged but not fatal: every remaining fanout unbind is still
// attempted and Run() still succeeds (double delivery beats loss).
func Test_Run_TopicMigration_FanoutUnbindFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	// Fail the unbind of the third fanout target; all 25 must still be attempted.
	failIndex := 2
	unbindAttempts := 0

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameSentinelEvent)).Return(nil)
	for _, target := range subscribeTargets {
		mockSock.EXPECT().QueueSubscribe(queueName, string(target)).Return(nil)
	}
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	for i, target := range fanoutUnbindTargets {
		mockSock.EXPECT().QueueUnbind(queueName, "", string(target), nil).
			DoAndReturn(func(_, _, _ string, _ interface{}) error {
				unbindAttempts++
				if i == failIndex {
					return fmt.Errorf("unbind failed")
				}
				return nil
			})
	}
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() must succeed when a fanout unbind fails (double delivery beats loss), got err: %v", err)
	}
	if unbindAttempts != len(fanoutUnbindTargets) {
		t.Errorf("Every fanout unbind must still be attempted after one failure. expect: %d, got: %d", len(fanoutUnbindTargets), unbindAttempts)
	}

	cancel()
	<-doneCh
}
