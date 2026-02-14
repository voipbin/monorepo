package subscribehandler

import (
	"fmt"
	"testing"

	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_Run_QueueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	h := &subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   "test-queue",
		subscribeTargets: []string{"target1"},
		tagHandler:       mockTag,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(fmt.Errorf("queue create failed"))

	err := h.Run()
	if err == nil {
		t.Errorf("Expected error when queue create fails, got nil")
	}
}

func Test_Run_QueueSubscribeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockTag := taghandler.NewMockTagHandler(mc)

	h := &subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   "test-queue",
		subscribeTargets: []string{"target1"},
		tagHandler:       mockTag,
	}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
	mockSock.EXPECT().QueueSubscribe("test-queue", "target1").Return(fmt.Errorf("subscribe failed"))

	err := h.Run()
	if err == nil {
		t.Errorf("Expected error when queue subscribe fails, got nil")
	}
}
