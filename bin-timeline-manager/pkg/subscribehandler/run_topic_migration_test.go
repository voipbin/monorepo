package subscribehandler

import (
	"context"
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/pkg/dbhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run_TopicMigrationSequence pins the full VOIP-1407 call sequence inside Run()
// with strict, ordered expectations:
//
//	QueueCreate -> asterisk fanout QueueSubscribe (retained leg) ->
//	VOIP-1258 webhook-topic QueueBind + QueueUnbind ->
//	TopicCreateWithKind(bin-manager.event) -> QueueBind("#") -> ConsumeMessage.
//
// ConsumeMessage runs on its own goroutine, so it is expected separately (AnyTimes)
// with an assertion that the synchronous startup sequence completed first.
func Test_Run_TopicMigrationSequence(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	var sequenceComplete bool

	gomock.InOrder(
		mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil),
		mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(nil),
		mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil),
		mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
		mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).
			DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
				sequenceComplete = true
				return nil
			}),
	)

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			if !sequenceComplete {
				t.Errorf("ConsumeMessage started before the topic-bind sequence completed -- ordering regression.")
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
	if !sequenceComplete {
		t.Errorf("Expected the topic-bind sequence to complete synchronously within Run().")
	}

	cancel()
	<-doneCh
}

// Test_Run_TopicMigration_DeclareFailure verifies that when the global topic exchange
// declare fails, Run() returns the error immediately (fatal -- there is no fanout
// fallback left to degrade to post-VOIP-1407).
func Test_Run_TopicMigration_DeclareFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("declare failed"))
	// NO QueueBind on bin-manager.event and NO ConsumeMessage: the strict mock
	// rejects any unexpected call, proving Run() returned immediately.

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err == nil {
		t.Fatalf("Run() must return an error immediately when the topic declare fails.")
	}
	if doneCh != nil {
		t.Errorf("Run() must return a nil channel alongside the error. got: %v", doneCh)
	}
}

// Test_Run_TopicMigration_BindFailure verifies that a pattern-bind failure returns the
// error immediately (fatal, no rollback -- §3.3).
func Test_Run_TopicMigration_BindFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("bind failed"))
	// NO rollback QueueUnbind on bin-manager.event and NO ConsumeMessage: the strict
	// mock rejects any such call, proving Run() returned immediately without rollback.

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err == nil {
		t.Fatalf("Run() must return an error immediately when the pattern bind fails.")
	}
	if doneCh != nil {
		t.Errorf("Run() must return a nil channel alongside the error. got: %v", doneCh)
	}
}

// Test_Run_AsteriskSubscribeFailure verifies that a failure subscribing to the
// retained asterisk fanout leg returns the error immediately (fatal).
func Test_Run_AsteriskSubscribeFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(fmt.Errorf("subscribe failed"))
	// NO further calls: the strict mock rejects any unexpected call, proving Run()
	// returned immediately.

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err == nil {
		t.Fatalf("Run() must return an error immediately when the asterisk fanout subscribe fails.")
	}
	if doneCh != nil {
		t.Errorf("Run() must return a nil channel alongside the error. got: %v", doneCh)
	}
}

// Test_Run_WebhookTopicBindFailure verifies the VOIP-1258 block's unchanged,
// log-only, non-fatal failure semantics: when the webhook-topic bind fails, Run()
// does NOT unbind the legacy fanout exchange, and Run() still succeeds.
func Test_Run_WebhookTopicBindFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(fmt.Errorf("bind failed"))
	// NO QueueUnbind of QueueNameWebhookEvent: the strict mock rejects it, proving
	// Run() stayed on the old exchange rather than unbinding after a failed bind.
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() must succeed when only the VOIP-1258 webhook-topic bind fails (log-only, non-fatal). err: %v", err)
	}

	cancel()
	<-doneCh
}

// Test_Run_WebhookLegacyUnbindFailure verifies the VOIP-1258 block's unchanged,
// log-only, non-fatal failure semantics on the unbind side: when the legacy fanout
// unbind fails after a successful topic bind, Run() still proceeds and succeeds.
func Test_Run_WebhookLegacyUnbindFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe(queueName, string(commonoutline.QueueNameAsteriskEventAll)).Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(fmt.Errorf("unbind failed"))
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() must succeed when only the VOIP-1258 legacy unbind fails (log-only, non-fatal). err: %v", err)
	}

	cancel()
	<-doneCh
}
