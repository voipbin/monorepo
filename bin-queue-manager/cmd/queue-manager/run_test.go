package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

func Test_runListen(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

	consumed := make(chan struct{})
	gomock.InOrder(
		mockSock.EXPECT().QueueCreate("bin-manager.queue-manager.request", "normal").Return(nil),
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "bin-manager.queue-manager.request", "queue-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				close(consumed)
				return nil
			}),
	)

	if err := runListen(mockSock, mockQueue, mockQueuecall); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Fatalf("the request consumer was not started")
	}
}

func Test_runSubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

	consumed := make(chan struct{})
	gomock.InOrder(
		mockSock.EXPECT().QueueCreate("bin-manager.queue-manager.subscribe", "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind("bin-manager.event", "topic").Return(nil),
		mockSock.EXPECT().QueueBind("bin-manager.queue-manager.subscribe", gomock.Any(), "bin-manager.event", false, nil).Return(nil).MinTimes(1),
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), "bin-manager.queue-manager.subscribe", "queue-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
				close(consumed)
				return nil
			}),
	)

	if err := runSubscribe(mockSock, mockQueue, mockQueuecall); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Fatalf("the event consumer was not started")
	}
}

func Test_runSubscribe_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

	errQueue := errors.New("simulated queue failure")
	// nothing else is expected after the failed queue creation.
	mockSock.EXPECT().QueueCreate("bin-manager.queue-manager.subscribe", "normal").Return(errQueue)

	if err := runSubscribe(mockSock, mockQueue, mockQueuecall); !errors.Is(err, errQueue) {
		t.Errorf("Wrong match. expect: %v, got: %v", errQueue, err)
	}
}
