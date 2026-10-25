package main

import (
	"errors"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

// Test_runListen_queueCreateFails records the current behavior: when the listen queue cannot be
// created, runListen only logs the failure and returns nil, and no consumer is started.
func Test_runListen_queueCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
	mockTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

	// ConsumeRPC has no expectation: a call fails the test
	mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameOutdialRequest), "normal").Return(errors.New("queue failure"))

	if err := runListen(mockSock, mockOutdial, mockTarget); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func Test_runListen_startsConsumer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
	mockTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

	queues := make(chan string, 1)
	mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameOutdialRequest), "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, queue, _ string, _, _, _ bool, _ int, _ any) error {
			queues <- queue
			return nil
		})

	if err := runListen(mockSock, mockOutdial, mockTarget); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}

	// the consumer runs in a goroutine, so the queue is asserted here
	select {
	case got := <-queues:
		if got != string(commonoutline.QueueNameOutdialRequest) {
			t.Errorf("expected the consumer on %q, got %q", commonoutline.QueueNameOutdialRequest, got)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the consumer was not started")
	}
}
