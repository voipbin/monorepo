package main

import (
	"context"
	"errors"
	"syscall"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/pkg/directhandler"
)

var errBrokerTest = errors.New("broker failure")

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameDirectRequest)

	t.Run("creates the request queue and starts the rpc consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consuming := make(chan struct{})
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "direct-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				close(consuming)
				return nil
			}).Times(1)

		if err := runListen(mockSock, directhandler.NewMockDirectHandler(mc)); err != nil {
			t.Fatalf("runListen() error = %v", err)
		}

		select {
		case <-consuming:
		case <-time.After(5 * time.Second):
			t.Fatalf("the rpc consumer was not started")
		}
	})

	t.Run("a queue failure does not stop runListen", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errBrokerTest)

		if err := runListen(mockSock, directhandler.NewMockDirectHandler(mc)); err != nil {
			t.Errorf("runListen() error = %v, want nil", err)
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameDirectSubscribe)

	t.Run("a queue failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errBrokerTest)

		err := runSubscribe(mockSock, directhandler.NewMockDirectHandler(mc))
		if !errors.Is(err, errBrokerTest) {
			t.Errorf("runSubscribe() error = %v, want %v wrapped", err, errBrokerTest)
		}
	})

	t.Run("binds the customer deleted topic and starts the consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consuming := make(chan struct{})
		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
			mockSock.EXPECT().QueueBind(queue, "customer-manager.customer.*.deleted", string(commonoutline.QueueNameEvent), false, nil).Return(nil),
		)
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, "direct-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
				close(consuming)
				return nil
			}).Times(1)

		if err := runSubscribe(mockSock, directhandler.NewMockDirectHandler(mc)); err != nil {
			t.Fatalf("runSubscribe() error = %v", err)
		}

		select {
		case <-consuming:
		case <-time.After(5 * time.Second):
			t.Fatalf("the event consumer was not started")
		}
	})
}

// Test_signalHandler_viaInit sends one value to chSigs. The goroutine that init() started
// with `go signalHandler()` consumes it and reports on chDone. The test never calls
// signalHandler itself, because that would race with the init goroutine for the same
// value, and it assumes -count=1 because the init goroutine consumes only one value per
// process. No other test in this package may use chSigs; the run tests use chDone only
// through run itself and drain it afterwards, and never run in parallel.
func Test_signalHandler_viaInit(t *testing.T) {
	chSigs <- syscall.SIGTERM

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("chDone = false, want true")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("a value sent to chSigs did not reach chDone")
	}
}
