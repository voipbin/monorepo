package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/pkg/messagehandler"
	"monorepo/bin-webchat-manager/pkg/sessionhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

// Test_runListen_queueCreateFails records the current behavior: the queue
// error is logged by the listen handler and runListen still returns nil.
func Test_runListen_queueCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	// no ConsumeRPC expectation: consuming after a failed queue creation fails the test
	mockSock.EXPECT().QueueCreate("bin-manager.webchat-manager.request", "normal").Return(errors.New("queue failure"))

	err := runListen(
		mockSock,
		widgethandler.NewMockWidgetHandler(mc),
		sessionhandler.NewMockSessionHandler(mc),
		messagehandler.NewMockMessageHandler(mc),
	)
	if err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

func Test_runListen_consumeStarted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	// ConsumeRPC runs in a goroutine started by the listen handler; the
	// buffered channel makes the test wait for it before the controller finishes.
	called := make(chan sock.CbMsgRPC, 1)
	mockSock.EXPECT().QueueCreate("bin-manager.webchat-manager.request", "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), "bin-manager.webchat-manager.request", "webchat-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, cb sock.CbMsgRPC) error {
			called <- cb
			return nil
		}).Times(1)

	err := runListen(
		mockSock,
		widgethandler.NewMockWidgetHandler(mc),
		sessionhandler.NewMockSessionHandler(mc),
		messagehandler.NewMockMessageHandler(mc),
	)
	if err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	select {
	case cb := <-called:
		if cb == nil {
			t.Errorf("The consume callback must not be nil.")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("ConsumeRPC was not called.")
	}
}

// Test_initSignal drives the handler through the package channel instead of a
// real process signal. initSignal is called once per test run and the channel
// is drained before the signal registration is removed.
func Test_initSignal(t *testing.T) {
	t.Cleanup(func() { signal.Stop(chSigs) })

	initSignal()
	chSigs <- syscall.SIGTERM

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("Wrong value. expect: true, got: false")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("chDone was not signalled.")
	}
}
