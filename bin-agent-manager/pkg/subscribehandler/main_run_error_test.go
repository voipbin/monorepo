package subscribehandler

import (
	stderrors "errors"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

func Test_Run_QueueCreateFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	queueName := string(commonoutline.QueueNameAgentSubscribe)
	errBoom := stderrors.New("boom")

	// no other broker call is allowed after the queue creation fails.
	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(errBoom)

	h := NewSubscribeHandler(mockSock, queueName, nil)

	err := h.Run()
	if !stderrors.Is(err, errBoom) {
		t.Errorf("Wrong error. expect: wrapped boom, got: %v", err)
	}
}

// Test_Run_UnbindAndConsumeFailuresAreNotFatal verifies that a failed unbind from the old exchange
// and a failed consume are only logged, and Run still reports success.
func Test_Run_UnbindAndConsumeFailuresAreNotFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	queueName := string(commonoutline.QueueNameAgentSubscribe)
	errBoom := stderrors.New("boom")

	consumed := make(chan struct{})

	mockSock.EXPECT().QueueCreate(queueName, "normal").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, "#", string(commonoutline.QueueNameWebhookEventTopic), false, nil).Return(nil)
	mockSock.EXPECT().QueueUnbind(queueName, "", string(commonoutline.QueueNameWebhookEvent), nil).Return(errBoom)
	mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queueName, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).Times(len(topicPatterns))
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, string(commonoutline.ServiceNameAgentManager), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
			close(consumed)
			return errBoom
		}).Times(1)

	h := NewSubscribeHandler(mockSock, queueName, nil)

	if err := h.Run(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Errorf("ConsumeMessage was not called.")
	}
}
