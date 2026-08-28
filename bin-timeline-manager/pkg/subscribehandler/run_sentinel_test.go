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

// Test_Run_DeclaresSentinelTopicBeforeSubscribe verifies that Run() declares the
// sentinel-manager event exchange before binding to it. sentinel-manager is Kubernetes-only
// (it needs the Kubernetes API), so in non-Kubernetes deployments nothing else declares that
// exchange and the bind would otherwise fail with an AMQP 404, taking this service down at
// boot. The declare uses the same fanout/durable parameters sentinel-manager's own
// notifyhandler uses, so it is an idempotent no-op when sentinel-manager is deployed.
func Test_Run_DeclaresSentinelTopicBeforeSubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	topicCreated := false

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameSentinelEvent)).
		DoAndReturn(func(_ string) error {
			topicCreated = true
			return nil
		})
	for _, target := range subscribeTargets {
		mockSock.EXPECT().QueueSubscribe(queueName, string(target)).
			DoAndReturn(func(_, topic string) error {
				if topic == string(commonoutline.QueueNameSentinelEvent) && !topicCreated {
					t.Errorf("Subscribed to the sentinel target before the topic was declared.")
				}
				return nil
			})
	}
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(nil)
	// VOIP-1406 topic migration block.
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameEvent), false, nil).Return(nil)
	for _, target := range fanoutUnbindTargets {
		mockSock.EXPECT().QueueUnbind(queueName, "", string(target), nil).Return(nil)
	}
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, gomock.Any(), false, false, false, 10, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCh, err := h.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	if !topicCreated {
		t.Errorf("Expected TopicCreate to have been called for the sentinel target.")
	}

	cancel()
	<-doneCh
}

// Test_Run_SentinelTopicCreateFailure verifies that a failed sentinel exchange declare is
// returned to the caller rather than being swallowed, matching how a failed QueueSubscribe is
// handled in the same loop.
func Test_Run_SentinelTopicCreateFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	queueName := string(commonoutline.QueueNameTimelineSubscribe)

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().TopicCreate(string(commonoutline.QueueNameSentinelEvent)).Return(fmt.Errorf("could not create the topic"))
	mockSock.EXPECT().QueueSubscribe(queueName, gomock.Any()).Return(nil).AnyTimes()

	h := NewSubscribeHandler(mockSock, mockDB)

	doneCh, err := h.Run(context.Background())
	if err == nil {
		t.Error("Wrong match. expect: error, got: ok")
	}
	if doneCh != nil {
		t.Errorf("Wrong match. expect: nil doneCh, got: %v", doneCh)
	}
}
