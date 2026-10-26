package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"
)

var testQueueName = string(commonoutline.QueueNameTTSRequest)

// Test_runListenNormal_success verifies the queue is declared with the exact
// name and that it waits for ConsumeRPC to be reached before returning.
func Test_runListenNormal_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	done := make(chan struct{})
	mockSock.EXPECT().QueueCreate(testQueueName, "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), testQueueName, "tts-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_, _, _, _, _, _ any, _ any, _ any) error {
			close(done)
			return nil
		},
	)

	if err := runListenNormal(mockSock, nil, nil, nil); err != nil {
		t.Fatalf("Wrong match. expected: nil, got: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("ConsumeRPC was not reached")
	}
}

// Test_runListenNormal_queueCreateError verifies QueueCreate failure is wrapped.
func Test_runListenNormal_queueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(testQueueName, "normal").Return(errors.New("boom"))

	err := runListenNormal(mockSock, nil, nil, nil)
	if err == nil {
		t.Fatal("Wrong match. expected: error, got: nil")
	}
}

// Test_runListenPod_success verifies the pod queue name is exactly "<queue>.<podID>".
func Test_runListenPod_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	done := make(chan struct{})
	expectQueue := fmt.Sprintf("%s.%s", testQueueName, "pod-1")
	mockSock.EXPECT().QueueCreate(expectQueue, "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), expectQueue, "tts-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_, _, _, _, _, _ any, _ any, _ any) error {
			close(done)
			return nil
		},
	)

	if err := runListenPod(mockSock, nil, nil, nil, "pod-1"); err != nil {
		t.Fatalf("Wrong match. expected: nil, got: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("ConsumeRPC was not reached")
	}
}

// Test_runListenPod_queueCreateError verifies QueueCreate failure is wrapped.
func Test_runListenPod_queueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(fmt.Sprintf("%s.%s", testQueueName, "pod-1"), "normal").Return(errors.New("boom"))

	err := runListenPod(mockSock, nil, nil, nil, "pod-1")
	if err == nil {
		t.Fatal("Wrong match. expected: error, got: nil")
	}
}

// Test_runListen_panicsOnNormalFailure verifies runListen panics when the
// normal-mode listener fails to start.
func Test_runListen_panicsOnNormalFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(testQueueName, "normal").Return(errors.New("boom"))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Wrong match. expected: panic, got: none")
		}
	}()

	runListen(mockSock, nil, nil, nil, "pod-1")
}

// Test_runListen_panicsOnPodFailure verifies runListen panics when the
// pod-mode listener fails to start, after the normal-mode ConsumeRPC goroutine
// has been reached (so it does not leak into the next test).
func Test_runListen_panicsOnPodFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	normalConsumed := make(chan struct{})
	mockSock.EXPECT().QueueCreate(testQueueName, "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), testQueueName, "tts-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
		func(_, _, _, _, _, _ any, _ any, _ any) error {
			close(normalConsumed)
			return nil
		},
	)
	mockSock.EXPECT().QueueCreate(fmt.Sprintf("%s.%s", testQueueName, "pod-1"), "normal").Return(errors.New("boom"))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Wrong match. expected: panic, got: none")
		}

		select {
		case <-normalConsumed:
		case <-time.After(3 * time.Second):
			t.Fatal("ConsumeRPC was not reached")
		}
	}()

	runListen(mockSock, nil, nil, nil, "pod-1")
}
