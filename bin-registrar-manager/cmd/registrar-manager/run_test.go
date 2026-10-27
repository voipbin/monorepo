package main

import (
	"context"
	"errors"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

var errInjectedSock = errors.New("injected sock failure")

const caseWaitTimeout = 5 * time.Second

// waitClosed waits for the channel to be closed by a mocked call that runs in a goroutine
// of the code under test, so the test ends only after the call has been made.
func waitClosed(t *testing.T, done <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(caseWaitTimeout):
		t.Fatalf("Wrong match. %s was not called in time", what)
	}
}

func Test_runListen_queueCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameRegistrarRequest), "normal").Return(errInjectedSock)
	// no ConsumeRPC expectation: the consumer must not start after the failure

	// the listener failure is logged and not returned
	if err := runListen(mockSock, nil, nil, nil, nil, nil); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

func Test_runListen_startsConsumer(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	queue := string(commonoutline.QueueNameRegistrarRequest)
	done := make(chan struct{})

	mockSock := sockhandler.NewMockSockHandler(mc)
	gomock.InOrder(
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "registrar-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
				defer close(done)
				if cbConsume == nil {
					t.Errorf("Wrong match. expect: a request callback, got: nil")
				}
				return nil
			}),
	)

	if err := runListen(mockSock, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	waitClosed(t, done, "ConsumeRPC")
}

func Test_runSubscribe_failures(t *testing.T) {
	queue := string(commonoutline.QueueNameRegistrarSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name  string
		setup func(m *sockhandler.MockSockHandler)
	}{
		{
			name: "queue creation fails",
			setup: func(m *sockhandler.MockSockHandler) {
				m.EXPECT().QueueCreate(queue, "normal").Return(errInjectedSock)
			},
		},
		{
			name: "topic creation fails",
			setup: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queue, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(errInjectedSock),
				)
			},
		},
		{
			name: "the first binding fails",
			setup: func(m *sockhandler.MockSockHandler) {
				gomock.InOrder(
					m.EXPECT().QueueCreate(queue, "normal").Return(nil),
					m.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
					m.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(errInjectedSock),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.setup(mockSock)
			// no ConsumeMessage expectation: the consumer must not start after a failure

			err := runSubscribe(mockSock, nil, nil, nil)
			if !errors.Is(err, errInjectedSock) {
				t.Errorf("Wrong match. expect: %v, got: %v", errInjectedSock, err)
			}
		})
	}
}

func Test_runSubscribe_bindsAndConsumes(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	queue := string(commonoutline.QueueNameRegistrarSubscribe)
	exchange := string(commonoutline.QueueNameEvent)
	done := make(chan struct{})

	mockSock := sockhandler.NewMockSockHandler(mc)
	gomock.InOrder(
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
		// one call per subscribed pattern; the pattern list itself is pinned by the subscribehandler tests
		mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(nil).MinTimes(1),
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, string(commonoutline.ServiceNameRegistrarManager), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error {
				defer close(done)
				if messageConsume == nil {
					t.Errorf("Wrong match. expect: an event callback, got: nil")
				}
				return nil
			}),
	)

	if err := runSubscribe(mockSock, nil, nil, nil); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}
	waitClosed(t, done, "ConsumeMessage")
}
